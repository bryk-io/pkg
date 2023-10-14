package rpc

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/credentials"
)

// Provides the `credentials.PerRPCCredentials` interface for regular
// text tokens.
type authToken struct {
	kind  string
	value string
}

func (at authToken) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	ri, _ := credentials.RequestInfoFromContext(ctx)
	if err := credentials.CheckSecurityLevel(ri.AuthInfo, credentials.PrivacyAndIntegrity); err != nil {
		return nil, fmt.Errorf("invalid connection security level: %w", err)
	}
	return map[string]string{
		"authorization": at.kind + " " + at.value,
		"uri":           strings.Join(uri, ","),
	}, nil
}

func (at authToken) RequireTransportSecurity() bool {
	return true
}
