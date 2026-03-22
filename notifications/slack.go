package notifications

import (
	// Go Internal Packages
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	// Local Packages
	config "flowx/config"
)

type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackBlock struct {
	Type string    `json:"type"`
	Text SlackText `json:"text"`
}

type SlackPayload struct {
	Blocks []SlackBlock `json:"blocks"`
}

type SlackAlerter struct {
	client *http.Client
	config config.Slack
}

func NewSlackAlerter(k config.Slack) *SlackAlerter {
	client := &http.Client{Timeout: 5 * time.Second}
	return &SlackAlerter{client: client, config: k}
}

func FormatFields(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s: %s", k, fields[k])
	}
	return b.String()
}

func GetSlackText(fields map[string]string) string {
	return fmt.Sprintf("```%s```", FormatFields(fields))
}

func (s *SlackAlerter) Send(ctx context.Context, alert Alert) error {
	header := SlackBlock{
		Type: "header",
		Text: SlackText{Type: "plain_text", Text: alert.Title},
	}

	body := SlackBlock{
		Type: "section",
		Text: SlackText{
			Type: "mrkdwn",
			Text: GetSlackText(alert.Fields),
		},
	}

	p := SlackPayload{Blocks: []SlackBlock{header, body}}

	jsonPayload, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	bodyReader := bytes.NewReader(jsonPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.WebhookURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return nil
}
