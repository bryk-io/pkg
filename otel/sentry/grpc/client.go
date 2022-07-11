package sentrygrpc

import (
	"context"

	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.bryk.io/pkg/otel/sentry/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Client provides ready-to-use instrumentation primitives for gRPC clients.
func Client(rep apiErrors.Reporter) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return unaryClientInterceptor(rep), streamClientInterceptor(rep)
}

func unaryClientInterceptor(rep apiErrors.Reporter) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		callOpts ...grpc.CallOption,
	) error {
		// get existing request metadata
		rm, _ := metadata.FromOutgoingContext(ctx)
		md := rm.Copy()

		// start new operation
		name, tags := spanInfo(method)
		op := rep.Start(ctx, "grpc.client", internal.AsTransaction(name))
		op.Tags(tags)
		defer op.Finish()

		// propagate trace ID in context metadata
		md.Set("sentry-trace", op.TraceID())
		ctx = metadata.NewOutgoingContext(op.Context(), md)

		// process request
		reportEvent(op, event{id: 1, desc: msgSent, payload: req})
		err := invoker(ctx, method, req, reply, cc, callOpts...)
		reportEvent(op, event{id: 1, desc: msgRecv, payload: reply})

		// report any errors
		if err != nil {
			op.Status("error")
			op.Report(err)
		}
		return err
	}
}

func streamClientInterceptor(rep apiErrors.Reporter) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		callOpts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// get existing request metadata
		rm, _ := metadata.FromOutgoingContext(ctx)
		md := rm.Copy()

		// start new operation
		name, tags := spanInfo(method)
		op := rep.Start(ctx, "grpc.client", internal.AsTransaction(name))
		op.Tags(tags)

		// propagate trace ID in context metadata
		md.Set("sentry-trace", op.TraceID())
		ctx = metadata.NewOutgoingContext(op.Context(), md)

		// open stream and report any errors
		s, err := streamer(ctx, desc, cc, method, callOpts...)
		if err != nil {
			op.Status("error")
			op.Report(err)
			op.Finish()
			return s, err
		}

		// create wrapped stream handler
		stream := wrapClientStream(ctx, op, s, desc)
		go func() {
			// wait for stream completion in the background; catch errors
			if err := <-stream.done; err != nil {
				op.Status("error")
				op.Report(err)
			}
			close(stream.close) // manually terminate stream event processing
			op.Finish()
		}()
		return stream, nil
	}
}
