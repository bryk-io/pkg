package extras

import (
	"go.mongodb.org/mongo-driver/event"
	mt "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

// MongoDBMonitor returns a monitor instance to collect telemetry data when
// connecting to a MongoDB database. To use the monitor, simply set it in
// the 'Monitor' client parameter. For example:
//
//   opts := mongodb_options.Client()
//   opts.Monitor = MongoMonitor()
func MongoDBMonitor() *event.CommandMonitor {
	return mt.NewMonitor()
}
