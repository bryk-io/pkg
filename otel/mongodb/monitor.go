package otelmongodb

import (
	"go.mongodb.org/mongo-driver/event"
	mt "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

// Monitor returns a mongoDB "command monitor" event that can be used to
// collect telemetry data when connecting to a MongoDB database. To use
// the monitor, simply set it in the 'Monitor' client parameter.
//
// For example:
//
//	opts := mongodb_options.Client()
//	opts.Monitor = MongoMonitor()
func Monitor() *event.CommandMonitor {
	return mt.NewMonitor()
}
