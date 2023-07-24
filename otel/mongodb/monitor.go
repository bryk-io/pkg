package mongodb

import (
	"go.mongodb.org/mongo-driver/event"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

// NewMonitor creates a new mongodb event CommandMonitor.
func NewMonitor() *event.CommandMonitor {
	return otelmongo.NewMonitor(otelmongo.WithCommandAttributeDisabled(false))
}
