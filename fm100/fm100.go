package fm100

import (
	"github.com/le-yams/finemold100-gateway/hamqtt"
)

const (
	ClientName = "fm100-gateway"
)

var (
	deviceMAC  string
	deviceID   string
	deviceName = "FM100B"

	deviceConfigTopic = "homeassistant/device/" + deviceID + "/config"

	probesStateTopic = map[uint8]string{
		1: "fm100/" + deviceID + "/probe01/state",
		2: "fm100/" + deviceID + "/probe02/state",
		3: "fm100/" + deviceID + "/probe03/state",
		4: "fm100/" + deviceID + "/probe04/state",
	}

	probesChannel = map[uint8]chan *string{
		1: make(chan *string),
		2: make(chan *string),
		3: make(chan *string),
		4: make(chan *string),
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
    "ids":"` + deviceID + `",
    "name": "` + deviceName + `",
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
      "unique_id": "` + deviceID + probeID + `",
      "value_template":"{{ value_json.temperature}}",
      "state_class": "measurement",
      "state_topic": "` + "fm100/" + deviceID + "/probe" + probeID + "/state" + `"
    }
`
}

func PublishProbeValue(client *hamqtt.Client, probe uint8, value string) error {
	return client.Publish(probesStateTopic[probe], []byte(`{"temperature":`+value+`}`), false)
}
