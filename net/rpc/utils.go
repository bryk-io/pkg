package rpc

import (
	"context"
	"crypto/tls"
	"net"
	"strings"

	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.bryk.io/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GetAddress returns the IPv4 address for the specified network interface.
func GetAddress(networkInterface string) (string, error) {
	switch networkInterface {
	case NetworkInterfaceLocal:
		return "127.0.0.1", nil
	case NetworkInterfaceAll:
		return "", nil
	default:
		ip, err := GetInterfaceIP(networkInterface)
		if err != nil {
			return "", errors.Wrap(err, "failed to retrieve interface's IP")
		}
		return ip.To4().String(), nil
	}
}

// GetInterfaceIP returns the IP address for a given network interface.
func GetInterfaceIP(name string) (net.IP, error) {
	i, err := net.InterfaceByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "unknown interface")
	}

	ls, err := i.Addrs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load interface address(es)")
	}

	var ip net.IP
	for _, addr := range ls {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP
			}
		}
	}

	if ip == nil {
		err = errors.New("no IP address found for network interface")
	}
	return ip, err
}

// LoadCertificate provides a helper method to conveniently parse and existing
// certificate and corresponding private key.
func LoadCertificate(cert []byte, key []byte) (tls.Certificate, error) {
	c, err := tls.X509KeyPair(cert, key)
	return c, errors.Wrap(err, "failed to load key pair")
}

// ContextWithMetadata returns a context with the provided value set as metadata.
// Any existing metadata already present in the context will be preserved. Intended
// to be used for outgoing RPC calls.
func ContextWithMetadata(ctx context.Context, md metadata.MD) context.Context {
	for k := range md {
		if vals := md.Get(k); len(vals) > 0 {
			size := len(vals)
			nv := make([]string, size)
			for i := 0; i < size; i++ {
				nv[i] = strings.TrimSpace(vals[i])
			}
			md.Set(k, nv...)
		}
	}
	orig, _ := metadata.FromOutgoingContext(ctx)
	return metadata.NewOutgoingContext(ctx, metadata.Join(orig, md))
}

// MetadataFromContext extracts and return the metadata values available in the
// provided context.
func MetadataFromContext(ctx context.Context) metadata.MD {
	var list []metadata.MD
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		list = append(list, md)
	}
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		list = append(list, md)
	}
	if md, ok := gwRuntime.ServerMetadataFromContext(ctx); ok {
		list = append(list, md.HeaderMD)
		list = append(list, md.TrailerMD)
	}
	return metadata.Join(list...)
}

// GetAuthToken is helper function for extracting the "authorization" header from
// the gRPC metadata of the request. It expects the "authorization" header to be of
// a certain scheme (e.g. `basic`, `bearer`), in a case-insensitive format
// (see rfc2617, sec 1.2). If no such authorization is found, or the token is of wrong
// scheme, an error with gRPC status `Unauthenticated` is returned.
func GetAuthToken(ctx context.Context, scheme string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "no authorization token")
	}
	t := md.Get("authorization")
	if len(t) == 0 {
		return "", status.Errorf(codes.Unauthenticated, "no authorization token")
	}
	splits := strings.SplitN(t[0], " ", 2)
	if len(splits) < 2 {
		return "", status.Errorf(codes.Unauthenticated, "bad authorization token")
	}
	if !strings.EqualFold(splits[0], scheme) {
		return "", status.Errorf(codes.Unauthenticated, "bad token scheme")
	}
	return splits[1], nil
}
