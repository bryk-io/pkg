package rpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	healthV1 "google.golang.org/grpc/health/grpc_health_v1"
)

// HealthCheck is a function that can be used to report whether a service
// is able to handle incoming client requests or not. If an error is returned
// the service will be marked as unavailable and respond with a status code
// of `NOT_SERVING`.
type HealthCheck func(ctx context.Context, service string) error

type healthSvc struct {
	srv *Server
}

func (hs *healthSvc) ServerSetup(_ *grpc.Server) {
	healthV1.RegisterHealthServer(hs.srv.grpc, hs)
}

func (hs *healthSvc) Check(ctx context.Context, req *healthV1.HealthCheckRequest) (*healthV1.HealthCheckResponse, error) { // nolint: lll
	// status field should be set to `SERVING` or `NOT_SERVING` accordingly.
	status := healthV1.HealthCheckResponse_SERVING
	if err := hs.srv.healthCheck(ctx, req.Service); err != nil {
		status = healthV1.HealthCheckResponse_NOT_SERVING
	}
	return &healthV1.HealthCheckResponse{Status: status}, nil
}

func (hs *healthSvc) Watch(req *healthV1.HealthCheckRequest, stream healthV1.Health_WatchServer) error { // nolint: lll
	// initial health check
	res, err := hs.Check(stream.Context(), req)
	if err != nil {
		return err
	}

	// start a goroutine to send periodic updates to the client
	// and close the stream when the client cancels the request
	// or the health check fails.
	// nolint: lll
	go func(st healthV1.HealthCheckResponse_ServingStatus, r *healthV1.HealthCheckRequest, client healthV1.Health_WatchServer) {
		// previous status detected
		previousStatus := st

		// do periodic health checks (every minute)
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			// health check
			case <-ticker.C:
				res, err := hs.Check(client.Context(), r)
				if err != nil {
					return
				}
				// if status changed, send update to client
				if res.Status != previousStatus {
					previousStatus = res.Status
					if err := client.Send(res); err != nil {
						return
					}
				}
			// client canceled request
			case <-client.Context().Done():
				return
			}
		}
	}(res.Status, req, stream)
	return nil
}

func (hs *healthSvc) ServiceDesc() grpc.ServiceDesc {
	return healthV1.Health_ServiceDesc
}
