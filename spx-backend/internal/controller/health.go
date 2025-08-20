package controller

import (
	"context"
	"time"
)

// Health performs a health check.
func (ctrl *Controller) Health(ctx context.Context) (*HealthResult, error) {
	return &HealthResult{
		Status: "ok",
		Time:   time.Now().Format(time.RFC3339),
	}, nil
}
