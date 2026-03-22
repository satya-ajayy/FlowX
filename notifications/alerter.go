package notifications

import (
	// Go Internal Packages
	"context"
	"errors"

	// Local Packages
	config "flowx/config"
)

type Alert struct {
	Title  string
	Fields map[string]string
}

type Alerter interface {
	Send(ctx context.Context, alert Alert) error
}

func NewAlerter(_ context.Context, k config.Config) Alerter {
	if k.IsProdMode || k.Slack.SendAlertInDev {
		return NewSlackAlerter(k.Slack)
	}
	return NewDiscardAlerter()
}

type MultiAlerter struct {
	alerters []Alerter
}

func NewMultiAlerter(alerters ...Alerter) *MultiAlerter {
	return &MultiAlerter{alerters: alerters}
}

func (m *MultiAlerter) Send(ctx context.Context, alert Alert) error {
	var errs []error
	for _, a := range m.alerters {
		if err := a.Send(ctx, alert); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
