package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// NotificationEvent describes a notification to deliver (e.g. on task assignment).
type NotificationEvent struct {
	Type      string
	Recipient uuid.UUID
	TaskID    uuid.UUID
	Message   string
}

// Notifier delivers notifications. It returns an error so a delivery failure can
// roll back the surrounding transaction (the case study allows a mock notifier).
type Notifier interface {
	Notify(ctx context.Context, event NotificationEvent) error
}

// LogNotifier is the mock implementation that records the notification to the log.
type LogNotifier struct {
	logger zerolog.Logger
}

func NewLogNotifier(logger zerolog.Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

func (n *LogNotifier) Notify(_ context.Context, event NotificationEvent) error {
	n.logger.Info().
		Str("notification_type", event.Type).
		Str("recipient", event.Recipient.String()).
		Str("task_id", event.TaskID.String()).
		Str("message", event.Message).
		Msg("notification dispatched")
	return nil
}
