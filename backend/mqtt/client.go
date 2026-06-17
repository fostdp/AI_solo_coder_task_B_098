package mqtt

import (
	"beacon-system/config"
	"beacon-system/models"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client mqttlib.Client
	topic  string
}

func New(cfg *config.Config) (*Client, error) {
	opts := mqttlib.NewClientOptions()
	brokerURL := fmt.Sprintf("tcp://%s:%d", cfg.MQTTBroker, cfg.MQTTPort)
	opts.AddBroker(brokerURL)
	opts.SetClientID("beacon-alert-service")
	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(func(c mqttlib.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})
	opts.SetOnConnectHandler(func(c mqttlib.Client) {
		log.Println("MQTT client connected")
	})

	if cfg.MQTTUser != "" {
		opts.SetUsername(cfg.MQTTUser)
		opts.SetPassword(cfg.MQTTPass)
	}

	client := mqttlib.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	return &Client{
		client: client,
		topic:  cfg.MQTTTopic,
	}, nil
}

func (c *Client) PublishAlert(alert *models.Alert) error {
	alertData := map[string]interface{}{
		"id":          alert.ID,
		"alert_type":  alert.AlertType,
		"severity":    alert.Severity,
		"title":       alert.Title,
		"description": alert.Description,
		"beacon_id":   alert.BeaconID,
		"link_id":     alert.LinkID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(alertData)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	token := c.client.Publish(c.topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish alert: %w", token.Error())
	}

	log.Printf("Alert published to MQTT topic %s: %s", c.topic, alert.Title)
	return nil
}

func (c *Client) PublishSignalReception(signal *models.SignalReception) error {
	topic := fmt.Sprintf("%s/signal/%d/%d", c.topic, signal.FromBeaconID, signal.ToBeaconID)
	payload, err := json.Marshal(signal)
	if err != nil {
		return err
	}

	token := c.client.Publish(topic, 0, false, payload)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *Client) PublishSensorData(data *models.SensorData) error {
	topic := fmt.Sprintf("beacon/sensor/%d", data.BeaconID)
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	token := c.client.Publish(topic, 0, false, payload)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *Client) Disconnect() {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
		log.Println("MQTT client disconnected")
	}
}
