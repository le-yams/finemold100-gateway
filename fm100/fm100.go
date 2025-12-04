package fm100

import (
	"github.com/le-yams/finemold100-gateway/ble"
	"github.com/le-yams/finemold100-gateway/hamqtt"
	"tinygo.org/x/bluetooth"
)

const (
	ClientName = "fm100-gateway"
)

var (
	DeviceMAC string
	DeviceID  string

	bleDevice *bluetooth.Device

	DeviceName = "FM100B"

	deviceConfigTopic = "homeassistant/device/" + DeviceID + "/config"

	probesStateTopic = map[uint8]string{
		1: "fm100/" + DeviceID + "/probe01/state",
		2: "fm100/" + DeviceID + "/probe02/state",
		3: "fm100/" + DeviceID + "/probe03/state",
		4: "fm100/" + DeviceID + "/probe04/state",
	}

	probesID = map[uint8]string{
		1: "01",
		2: "02",
		3: "03",
		4: "04",
	}

	probesName = map[uint8]string{
		1: "probe-01",
		2: "probe-02",
		3: "probe-03",
		4: "probe-04",
	}

	deviceConfig = `{
  "dev": {
    "ids":"` + DeviceID + `",
    "name": "` + DeviceName + `",
  },
  "o": {
    "name": "` + ClientName + `",
  },
  "cmps": {
    ` + homeAssistantProbeConfig(1) + `,
    ` + homeAssistantProbeConfig(2) + `,
    ` + homeAssistantProbeConfig(3) + `,
    ` + homeAssistantProbeConfig(4) + `
  }
}`
)

func PublishDeviceConfig(client *hamqtt.Client) error {
	err := client.Publish(deviceConfigTopic, []byte(deviceConfig), true)
	if err != nil {
		return err
	}
	return err
}

func homeAssistantProbeConfig(probe uint8) string {
	probeID := probesID[probe]
	probeName := probesName[probe]
	return `"0x544E94C09BC8_` + probeName + `": {
      "name": "` + probeName + `",
      "p": "sensor",
      "device_class": "temperature",
      "unit_of_measurement": "°C",
      "unique_id": "` + DeviceID + probeID + `",
      "value_template":"{{ value_json.temperature}}",
      "state_class": "measurement",
      "state_topic": "` + "fm100/" + DeviceID + "/probe" + probeID + "/state" + `"
    }
`
}

func PublishProbeValue(client *hamqtt.Client, probe uint8, value string) error {
	return client.Publish(probesStateTopic[probe], []byte(`{"temperature":`+value+`}`), false)
}

var bleServiceUUIDName = bluetooth.New16BitUUID(0x1800)
var bleCharacteristicName = bluetooth.New16BitUUID(0x2A00)

var bleServiceUUIDInfo = bluetooth.New16BitUUID(0x180A)
var bleCharacteristicModel = bluetooth.New16BitUUID(0x2A24)
var bleCharacteristicSerial = bluetooth.New16BitUUID(0x2A25)
var bleCharacteristicHardware = bluetooth.New16BitUUID(0x2A27)
var bleCharacteristicSoftware = bluetooth.New16BitUUID(0x2A28)
var bleCharacteristicManufacturer = bluetooth.New16BitUUID(0x2A29)

var bleServiceUUIDThermo = bluetooth.New16BitUUID(0xFF00)
var bleCharacteristicNotify = bluetooth.New16BitUUID(0xFF01)
var bleCharacteristicWrite = bluetooth.New16BitUUID(0xFF01)

func ConnectBLE(client *hamqtt.Client) error {
	adapter := bluetooth.DefaultAdapter
	err := adapter.Enable()
	if err != nil {
		return err
	}

	macAddress := bluetooth.MACAddress{}
	macAddress.Set(DeviceMAC)

	adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		bleDevice = &device
		status := "disconnected"
		if connected {
			println("BLE device connection handler for device", bleDevice.Address.String())
			onDeviceConnect()
		}
		println("BLE Device", status)
	})

	println("connecting to BLE device:", DeviceMAC)
	_, err = adapter.Connect(bluetooth.Address{MACAddress: macAddress}, bluetooth.ConnectionParams{})

	return err
}

func onDeviceConnect() {
	chars, err := ble.GetCharacteristics(bleDevice, bleServiceUUIDName, bleCharacteristicName)
	if err != nil {
		println("could not get device name characteristic", err.Error())
		return
	}
	println("found", len(chars), "characteristics")
	for i, c := range chars {
		if c.UUID().String() == bleCharacteristicName.String() {
			println("reading characteristic", i)
			name, err := ble.ReadCharacteristicAsString(c)
			if err != nil {
				println("could not read device name characteristic", err.Error())
				return
			}
			println("connected to device:", name)
		} else {
			println("skipping characteristic", i, "with UUID", c.UUID().String())
		}

	}

	chars, err = ble.GetCharacteristics(bleDevice, bleServiceUUIDInfo, bleCharacteristicModel, bleCharacteristicSerial, bleCharacteristicHardware, bleCharacteristicSoftware, bleCharacteristicManufacturer)
	if err != nil {
		println("could not get device info characteristics", err.Error())
		return
	}
	println("found", len(chars), "characteristics")
	for _, c := range chars {
		println("reading characteristic", c.UUID().String())
		value, err := ble.ReadCharacteristicAsString(c)
		if err != nil {
			println("could not read device info characteristic", err.Error())
			return
		}

		println("device info:", value)

	}

	chars, err = ble.GetCharacteristics(bleDevice, bleServiceUUIDThermo, bleCharacteristicNotify)
	if err != nil {
		println("could not get thermo characteristics", err.Error())
		return
	}
	println("found", len(chars), "characteristics in thermo service")
	for i, c := range chars {
		if c.UUID().String() == bleCharacteristicNotify.String() {
			println("subscribing to characteristic", c.UUID().String())
			err = c.EnableNotifications(onThermoNotification)
			if err != nil {
				println("could not subscribe to thermo notification characteristic", err.Error())
				return
			}
			println("subscribed to thermo notifications")
		} else {
			println("skipping characteristic", i, "with UUID", c.UUID().String())
		}

	}
}

func onThermoNotification(value []byte) {
	println("received thermo notification:", value)
}
