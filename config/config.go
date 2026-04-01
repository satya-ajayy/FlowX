package config

import (
	// Go Internal Packages
	"fmt"

	// Local Packages
	errors "flowx/errors"
	flow "flowx/flow"
	helpers "flowx/utils/helpers"
)

var DefaultConfig = []byte(`
application: "flowx"

logger:
  encoding: "logfmt"
  level: "debug"

listen: ":3625"

prefix: "/flowx"

is_prod_mode: false

mongo:
  uri: "mongodb://localhost:27017"

queue:
  size: 50
  workers: 5

executor:
  flow: "default"
  max_retries: 3
  initial_backoff: 30
  max_backoff: 300
  backoff_factor: 2.0
  jitter_fraction: 0.2

slack:
  webhook_url: "https://hooks.slack.com/services/your/webhook/url"
  send_alert_in_dev: false
`)

type Config struct {
	Application string   `koanf:"application"`
	Logger      Logger   `koanf:"logger"`
	Listen      string   `koanf:"listen"`
	Prefix      string   `koanf:"prefix"`
	IsProdMode  bool     `koanf:"is_prod_mode"`
	Mongo       Mongo    `koanf:"mongo"`
	Queue       Queue    `koanf:"queue"`
	Executor    Executor `koanf:"executor"`
	Slack       Slack    `koanf:"slack"`
}

type Logger struct {
	Encoding string `koanf:"encoding"`
	Level    string `koanf:"level"`
}

type Mongo struct {
	URI string `koanf:"uri"`
}

type Queue struct {
	Size    int `koanf:"size"`
	Workers int `koanf:"workers"`
}

// Executor is the configuration for the executor service.
// Backoff durations are in seconds in YAML and converted to time.Duration.
type Executor struct {
	Flow           string  `koanf:"flow"`
	MaxRetries     int     `koanf:"max_retries"`
	InitialBackoff int     `koanf:"initial_backoff"` // seconds
	MaxBackoff     int     `koanf:"max_backoff"`     // seconds
	BackoffFactor  float64 `koanf:"backoff_factor"`
	JitterFraction float64 `koanf:"jitter_fraction"` // 0.0 to 1.0
}

type Endpoint struct {
	URL string `koanf:"url"`
}

type Slack struct {
	WebhookURL     string `koanf:"webhook_url"`
	SendAlertInDev bool   `koanf:"send_alert_in_dev"`
}

// Validate checks all required configuration fields.
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
	helpers.ValidateRequiredNumber(ve, "queue.size", c.Queue.Size)
	helpers.ValidateRequiredNumber(ve, "queue.workers", c.Queue.Workers)

	// Executor Fields
	helpers.ValidateRequiredString(ve, "executor.flow", c.Executor.Flow)
	helpers.ValidateRequiredNumber(ve, "executor.max_retries", c.Executor.MaxRetries)
	helpers.ValidateRequiredNumber(ve, "executor.initial_backoff", c.Executor.InitialBackoff)
	helpers.ValidateRequiredNumber(ve, "executor.max_backoff", c.Executor.MaxBackoff)

	if !flow.Exists(c.Executor.Flow) {
		ve.Add("executor.flow", fmt.Sprintf("%s not found in registry:", c.Executor.Flow))
	}
	if c.Executor.BackoffFactor < 1.0 {
		ve.Add("executor.backoff_factor", "must be >= 1.0")
	}
	if c.Executor.JitterFraction < 0 || c.Executor.JitterFraction > 1.0 {
		ve.Add("executor.jitter_fraction", "must be between 0 and 1")
	}

	return ve.Err()
}
