package sentrygrpc

import (
	"context"
	"net"

	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.bryk.io/pkg/otel/sentry/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Server provides ready-to-use instrumentation primitives for gRPC servers.
func Server(rep apiErrors.Reporter) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return unaryServerInterceptor(rep), streamServerInterceptor(rep)
}

func unaryServerInterceptor(rep apiErrors.Reporter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		// get existing request metadata
		rm, _ := metadata.FromIncomingContext(ctx)
		md := rm.Copy()

		// get operation settings
		name, tags := spanInfo(info.FullMethod)
		opts := []apiErrors.OperationOption{internal.AsTransaction(name)}
		if v := md.Get("sentry-trace"); len(v) == 1 {
			opts = append(opts, internal.ToContinue(v[0]))
		}

		// start new operation
		op := rep.Start(ctx, "grpc.server", opts...)
		op.Tags(tags)
		setOperationDetails(ctx, op)
		defer op.Finish()

		// propagate operation in context
		ctx = rep.ToContext(ctx, op)

		// process request
		reportEvent(op, event{id: 1, desc: msgRecv, payload: req})
		resp, err = handler(ctx, req)

		// report any errors
		if err != nil {
			op.Status("error")
			op.Report(err)
			s, _ := status.FromError(err)
			reportEvent(op, event{id: 1, desc: msgSent, payload: s})
		} else {
			reportEvent(op, event{id: 1, desc: msgSent, payload: resp})
		}
		return resp, err
	}
}

func streamServerInterceptor(rep apiErrors.Reporter) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// get existing request metadata
		ctx := ss.Context()
		rm, _ := metadata.FromIncomingContext(ctx)
		md := rm.Copy()

		// get operation settings
		name, tags := spanInfo(info.FullMethod)
		opts := []apiErrors.OperationOption{internal.AsTransaction(name)}
		if v := md.Get("sentry-trace"); len(v) == 1 {
			opts = append(opts, internal.ToContinue(v[0]))
		}

		// start new operation
		op := rep.Start(ctx, "grpc.server", opts...)
		op.Tags(tags)
		setOperationDetails(ctx, op)
		defer op.Finish()

		// create wrapped stream handler
		err := handler(srv, wrapServerStream(ctx, op, ss))
		if err != nil {
			op.Status("error")
			op.Report(err)
		}
		return err
	}
}

func setOperationDetails(ctx context.Context, op apiErrors.Operation) {
	io, ok := op.(*internal.Operation)
	if !ok {
		return
	}

	// Get peer IP
	p, ok := peer.FromContext(ctx)
	if !ok {
		return
	}
	ip, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return
	}

	// Look for proxy protocol details propagated as metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if addr := md.Get("x-forwarded-for"); len(addr) > 0 {
			ip = addr[0]
		}
		if addr := md.Get("x-real-ip"); len(addr) > 0 {
			ip = addr[0]
		}
	}

	// Set user IP on operation
	if ip != "" && ip != "127.0.0.1" {
		io.User(apiErrors.User{IPAddress: ip})
	}
}
