package config

import (
	// Local Packages
	errors "flowx/errors"
)

var DefaultConfig = []byte(`
application: "flowx"

logger:
  level: "debug"

listen: ":3625"

prefix: "/flowx"

is_prod_mode: false

mongo:
  uri: "mongodb://localhost:27017"

queue:
  size: 50
  workers: 5

slack:
 webhook_url: "https://hooks.slack.com/services/your/webhook/url"
 send_alert_in_dev: false
`)

type Config struct {
	Application string `koanf:"application"`
	Logger      Logger `koanf:"logger"`
	Listen      string `koanf:"listen"`
	Prefix      string `koanf:"prefix"`
	IsProdMode  bool   `koanf:"is_prod_mode"`
	Mongo       Mongo  `koanf:"mongo"`
	Queue       Queue  `koanf:"queue"`
	Slack       Slack  `koanf:"slack"`
}

type Logger struct {
	Level string `koanf:"level"`
}

type Mongo struct {
	URI string `koanf:"uri"`
}

type Queue struct {
	Size    int `koanf:"size"`
	Workers int `koanf:"workers"`
}

type Endpoint struct {
	URL string `koanf:"url"`
}

type Slack struct {
	WebhookURL     string `koanf:"webhook_url"`
	SendAlertInDev bool   `koanf:"send_alert_in_dev"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	ve := errors.ValidationErrs()

	if c.Application == "" {
		ve.Add("application", "cannot be empty")
	}
	if c.Listen == "" {
		ve.Add("listen", "cannot be empty")
	}
	if c.Logger.Level == "" {
		ve.Add("logger.level", "cannot be empty")
	}
	if c.Prefix == "" {
		ve.Add("prefix", "cannot be empty")
	}
	if c.Mongo.URI == "" {
		ve.Add("mongo.uri", "cannot be empty")
	}
	if c.Slack.WebhookURL == "" {
		ve.Add("slack.webhook_url", "cannot be empty")
	}
	if c.Queue.Size == 0 {
		ve.Add("queue.size", "cannot be empty")
	}
	if c.Queue.Workers == 0 {
		ve.Add("queue.workers", "cannot be empty")
	}

	return ve.Err()
}
