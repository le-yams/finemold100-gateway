package main

import (
	"machine"
	"math/rand"
	"strconv"
	"time"

	"github.com/le-yams/finemold100-gateway/fm100"
	"github.com/le-yams/finemold100-gateway/hamqtt"
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

	mqttStatus   = 0
	mqttServer   string
	mqttUsername string
	mqttPassword string
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
	client, err := hamqtt.Connect(fm100.ClientName, mqttServer, mqttUsername, mqttPassword, 4*time.Second)
	mqttStatus = 1
	if err != nil {
		led.Low()
		println("could not initialize MQTT client:", err.Error())
		return
	}
	println("connected to MQTT broker", mqttServer)

	err = fm100.PublishDeviceConfig(client)
	if err != nil {
		println("could not publish device config:", err.Error())
		return
	}

	publishSomeFakeData(client)

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

func publishSomeFakeData(client *hamqtt.Client) {

	// Simulate FM100 device, in real implementation read from BLE
	loops := 0
	i := 0
	probeNumber := uint8(0)
	time.Sleep(5 * time.Second)
	println("starting loop to fake probe data")
	for loops < 10 {
		probeNumber = (probeNumber % 4) + 1

		d := 20 + rand.Intn(30)
		i = (i + 1) % 10

		v := strconv.Itoa(d) + "." + strconv.Itoa(i)

		println("faking probe", probeNumber, "data:", v)

		err := fm100.PublishProbeValue(client, probeNumber, v)
		if err != nil {
			println("could not publish probe", probeNumber, " value:", err.Error())
		}

		time.Sleep(2 * time.Second)
		loops++
	}
}
