package hamqtt

import (
	"context"
	"net"
	"time"

	"github.com/soypat/natiu-mqtt"
)

type Client mqtt.Client

func Connect(clientName, server, username, password string, timeout time.Duration) (*Client, error) {
	client := mqtt.NewClient(mqtt.ClientConfig{})
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}
	var varConn mqtt.VariablesConnect
	varConn.SetDefaultMQTT([]byte(clientName))
	if username != "" {
		varConn.Username = []byte(username)
	}
	if password != "" {
		varConn.Password = []byte(password)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	err = client.Connect(ctx, conn, &varConn)
	cancel()

	return (*Client)(client), err
}

func (c *Client) mqtt() *mqtt.Client {
	return (*mqtt.Client)(c)
}

func (c *Client) Publish(topic string, payload []byte, retained bool) error {
	flags, _ := mqtt.NewPublishFlags(mqtt.QoS0, false, retained)
	variables := mqtt.VariablesPublish{
		TopicName:        []byte(topic),
		PacketIdentifier: 1,
	}
	return c.mqtt().PublishPayload(flags, variables, payload)
}
