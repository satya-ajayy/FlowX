package notifications

import (
	// Go Internal Packages
	"context"
)

type DiscardAlerter struct{}

func NewDiscardAlerter() *DiscardAlerter {
	return &DiscardAlerter{}
}

func (d *DiscardAlerter) Send(_ context.Context, _ Alert) error {
	return nil
}
