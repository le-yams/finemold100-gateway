package main

import (
	"context"
	"machine"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/soypat/natiu-mqtt"
	"tinygo.org/x/drivers/netlink"
	"tinygo.org/x/drivers/netlink/probe"
)

var (
	ssid     string
	password string

	uart = machine.UART1
	tx   = machine.PA22
	rx   = machine.PA23

	wifiStatus = 0

	fm100DeviceMAC  string
	fm100DeviceName = "FM100B" // to be read from ble

	mqttStatus            = 0
	mqttServer            string
	mqttUsername          string
	mqttPassword          string
	mqttClientName        = "FM100-gateway"
	mqttDeviceID          string
	mqttDeviceConfigTopic string
)

func main() {
	// Wait a bit for console
	time.Sleep(2 * time.Second)

	// Configure UART
	err := uart.Configure(machine.UARTConfig{TX: tx, RX: rx})
	if err != nil {
		println(err.Error())
		return
	}
	led := machine.LED
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})

	// Connecting to Wi-Fi
	ledBlinkWhile(led, func() bool { return wifiStatus == 0 }, 500*time.Millisecond, 500*time.Millisecond)
	link, _ := probe.Probe()
	err = link.NetConnect(&netlink.ConnectParams{
		Ssid:       ssid,
		Passphrase: password,
	})
	if err != nil {
		wifiStatus = -1
		println(err.Error())
		return
	}
	wifiStatus = 1
	println("connected to wifi network:", ssid)

	// Connecting to MQTT broker
	ledBlinkWhile(led, func() bool { return mqttStatus == 0 }, 200*time.Millisecond, 200*time.Millisecond)
	client, err := initMQTTClient()
	mqttStatus = 1
	if err != nil {
		led.Low()
		println("could not initialize MQTT client:", err.Error())
		return
	}
	println("connected to MQTT server", mqttServer)

	// Simulate FM100 device, in real implementation read from BLE
	err = publishDeviceConfig(client)
	if err != nil {
		println("could not publish device config:", err.Error())
	}

	i := 0
	loops := 0
	p := uint8(0)
	for loops < 30 {
		p = (p % 4) + 1
		println("faking probe", p, "data...")
		d := 20 + rand.Intn(10)
		i = (i + 1) % 10
		v := strconv.Itoa(d) + "." + strconv.Itoa(i)

		publishProbeValue(client, p, v)

		time.Sleep(2 * time.Second)
		loops++
	}
	println("done.")
	led.Low()
}

func ledBlinkWhile(led machine.Pin, condition func() bool, highDelay time.Duration, lowDelay time.Duration) {
	go func() {
		for condition() {
			led.High()
			time.Sleep(highDelay)
			led.Low()
			time.Sleep(lowDelay)
		}
		led.High()
	}()
}

func initMQTTClient() (*mqtt.Client, error) {
	mqttDeviceID = "0x" + strings.ReplaceAll(fm100DeviceMAC, ":", "")
	mqttDeviceConfigTopic = "homeassistant/device/" + mqttDeviceID + "/config"

	println("using device ID:", mqttDeviceID)
	client := mqtt.NewClient(mqtt.ClientConfig{})

	conn, err := net.Dial("tcp", mqttServer)
	if err != nil {
		return nil, err
	}
	var varConn mqtt.VariablesConnect
	varConn.SetDefaultMQTT([]byte("fm100-gateway"))
	if mqttUsername != "" {
		varConn.Username = []byte(mqttUsername)
	}
	if mqttPassword != "" {
		varConn.Password = []byte(mqttPassword)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	err = client.Connect(ctx, conn, &varConn)
	cancel()

	return client, err
}

func publishProbeValue(client *mqtt.Client, probeID uint8, value string) {
	println("publishing probe", probeID, "value:", value)
	err := publishProbeState(client, probeID, value)
	if err != nil {
		println("could not publish probe", probeID, "value:", err.Error())
	}
}

func publishDeviceConfig(client *mqtt.Client) error {
	flags, _ := mqtt.NewPublishFlags(mqtt.QoS0, false, true)
	variables := mqtt.VariablesPublish{
		TopicName:        []byte(mqttDeviceConfigTopic),
		PacketIdentifier: 1,
	}
	payload := []byte(deviceConfigPayload())
	return client.PublishPayload(flags, variables, payload)
}

func publishProbeState(client *mqtt.Client, probeNumber uint8, temperature string) error {
	flags, _ := mqtt.NewPublishFlags(mqtt.QoS0, false, false)
	variables := mqtt.VariablesPublish{
		TopicName:        []byte(probeStateTopic(getProbeID(probeNumber))),
		PacketIdentifier: 1,
	}
	payload := []byte(`{"temperature":` + temperature + `}`)
	return client.PublishPayload(flags, variables, payload)
}

func deviceConfigPayload() string {
	return `{
  "dev": {
    "ids":"` + mqttDeviceID + `",
    "name": "` + fm100DeviceName + `",
  },
  "o": {
    "name": "` + fm100DeviceName + `",
  },
  "cmps": {
    ` + probeConfigPayload(1) + `,
    ` + probeConfigPayload(2) + `,
    ` + probeConfigPayload(3) + `,
    ` + probeConfigPayload(4) + `
  }
}`
}

func probeConfigPayload(probeNumber uint8) string {
	probeID := getProbeID(probeNumber)
	probeName := "probe-" + probeID
	return `"0x544E94C09BC8_` + probeName + `": {
      "name": "` + probeName + `",
      "p": "sensor",
      "device_class": "temperature",
      "unit_of_measurement": "°C",
      "unique_id": "` + mqttDeviceID + probeID + `",
      "value_template":"{{ value_json.temperature}}",
      "state_class": "measurement",
      "state_topic": "` + probeStateTopic(probeID) + `"
    }
`
}

func getProbeID(probeNumber uint8) string {
	return "0" + strconv.Itoa(int(probeNumber))
}

func probeStateTopic(probeID string) string {
	return "fm100/" + mqttDeviceID + "/probe" + probeID + "/state"
}
