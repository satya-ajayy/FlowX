package config

import (
	// Local Packages
	errors "flowx/errors"
	helpers "flowx/utils/helpers"
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
// Validate validates the configuration
func (c *Config) Validate() error {
	ve := errors.ValidationErrs()

	// Required String Fields
	helpers.ValidateRequiredString(ve, "application", c.Application)
	helpers.ValidateRequiredString(ve, "listen", c.Listen)
	helpers.ValidateRequiredString(ve, "logger.level", c.Logger.Level)
	helpers.ValidateRequiredString(ve, "prefix", c.Prefix)
	helpers.ValidateRequiredString(ve, "mongo.uri", c.Mongo.URI)
	helpers.ValidateRequiredString(ve, "slack.webhook_url", c.Slack.WebhookURL)

	// Required Numeric Fields
	helpers.ValidateRequiredNumeric(ve, "queue.size", c.Queue.Size)
	helpers.ValidateRequiredNumeric(ve, "queue.workers", c.Queue.Workers)

	return ve.Err()
}
