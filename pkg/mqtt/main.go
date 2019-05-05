package mqtt

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/viart/device-locator/pkg/fmip"
)

type Cfg struct {
	Broker   string
	ID       string
	Username string
	Password string
	LWT      string
	Preffix  string
}

type MqttClient struct {
	mqtt.Client
	Cfg
}

func New(cfg Cfg) (*MqttClient, error) {
	opts := mqtt.NewClientOptions().AddBroker(cfg.Broker).SetClientID(cfg.ID).SetAutoReconnect(true)
	if cfg.LWT != "" {
		opts.SetBinaryWill(cfg.LWT, []byte("0"), 1, true)
		opts.SetOnConnectHandler(func(c mqtt.Client) {
			c.Publish(cfg.LWT, 1, true, []byte("1"))
		})
	}
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		if cfg.Password != "" {
			opts.SetPassword(cfg.Password)
		}
	}

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MqttClient{
		c,
		cfg,
	}, nil
}

func (m *MqttClient) Track(acc string, res *fmip.FmipResponse) {
	for _, device := range res.Content {
		data := map[string]interface{}{
			"_type": "location",
			"tst":   time.Now().Unix(),
			"name":  device.Name,
			"vac":   device.Location.VerticalAccuracy,
			"acc":   device.Location.HorizontalAccuracy,
			"lat":   device.Location.Latitude,
			"lon":   device.Location.Longitude,
			"alt":   device.Location.Altitude,
			"batt":  device.BatteryLevel * 100,
		}

		//TODO: handle error
		if payload, err := json.Marshal(data); err == nil {
			topic := fmt.Sprintf("%s/%s/%s", m.Cfg.Preffix, acc, strings.ReplaceAll(device.DeviceDisplayName, "-", ""))
			m.Publish(topic, 1, false, payload)
		}
	}
}
