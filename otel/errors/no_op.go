package errors

import (
	"context"
	"time"
)

// NoOpReporter provides a "dummy" reporter instance that simply discards
// error data.
func NoOpReporter() Reporter {
	return new(noOpReporter)
}

// NoOpOperation returns an element compliant with the `Operation` interface
// but that collects no data and produces no output. This is particularly useful
// to dynamically enable/disable instrumentation without modifying the code.
func NoOpOperation() Operation {
	return new(noOp)
}

// No-op reporter.
type noOpReporter struct{}

func (nr *noOpReporter) Inject(_ Operation, _ Carrier)                              {}
func (nr *noOpReporter) Extract(ctx context.Context, _ Carrier) context.Context     { return ctx }
func (nr *noOpReporter) Flush(_ time.Duration) bool                                 { return true }
func (nr *noOpReporter) ToContext(ctx context.Context, _ Operation) context.Context { return ctx }
func (nr *noOpReporter) FromContext(_ context.Context) Operation                    { return new(noOp) }
func (nr *noOpReporter) Start(_ context.Context, _ string, _ ...OperationOption) Operation {
	return new(noOp)
}

// No-op operation.
type noOp struct{}

func (n *noOp) Level(_ string)                              {}
func (n *noOp) Tags(_ map[string]interface{})               {}
func (n *noOp) Segment(_ string, _ interface{})             {}
func (n *noOp) Event(_ string, _ ...map[string]interface{}) {}
func (n *noOp) Context() context.Context                    { return context.Background() }
func (n *noOp) Report(_ error) string                       { return "" }
func (n *noOp) Finish()                                     {}
func (n *noOp) TraceID() string                             { return "" }
func (n *noOp) Status(_ string)                             {}
func (n *noOp) Inject(_ Carrier)                            {}
func (n *noOp) User(_ User)                                 {}
