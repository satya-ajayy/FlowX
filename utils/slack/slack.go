package slack

import (
	// Go Internal Packages
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	// Local Packages
	config "flowx/config"
)

// Alert represents a structured notification to be sent to Slack.
type Alert struct {
	Title  string
	Fields map[string]string
}

// Sender is the interface for sending alerts. This abstraction allows
// swapping Slack for other notification backends or a no-op in tests.
type Sender interface {
	Send(ctx context.Context, alert Alert) error
}

type slackSender struct {
	client *http.Client
	config config.Slack
	isProd bool
}

// NewSender creates a new Slack alert sender.
func NewSender(cfg config.Slack, isProd bool) Sender {
	return &slackSender{
		client: &http.Client{Timeout: 5 * time.Second},
		config: cfg,
		isProd: isProd,
	}
}

// Send dispatches an alert to Slack. In non-prod mode, alerts are silently
// discarded unless SendAlertInDev is explicitly enabled in config.
func (s *slackSender) Send(ctx context.Context, alert Alert) error {
	if !s.isProd && !s.config.SendAlertInDev {
		return nil
	}

	header := block{
		Type: "header",
		Text: text{Type: "plain_text", Text: alert.Title},
	}

	var lines []string
	for k, v := range alert.Fields {
		lines = append(lines, fmt.Sprintf("%s: %s", k, v))
	}

	body := block{
		Type: "section",
		Text: text{Type: "mrkdwn", Text: fmt.Sprintf("```%s```", strings.Join(lines, "\n"))},
	}

	payload := payload{Blocks: []block{header, body}}
	jsonPayload, _ := json.Marshal(payload)

	_, err := s.client.Post(s.config.WebhookURL, "application/json", bytes.NewReader(jsonPayload))
	return err
}

// Slack block-kit primitives (unexported — implementation detail).
type text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type block struct {
	Type string `json:"type"`
	Text text   `json:"text"`
}

type payload struct {
	Blocks []block `json:"blocks"`
}
