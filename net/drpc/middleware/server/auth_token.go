package server

import (
	"errors"

	"storj.io/drpc"
	"storj.io/drpc/drpcmetadata"
)

// TokenValidator represents an external authentication mechanism used to validate
// bearer credentials. In case of any error the validation function must return
// 'false' and the server will return an 'invalid credentials' error message.
type TokenValidator func(token string) bool

// AuthByToken allows to use an external authentication mechanism using bearer
// tokens as credentials. The token must be present in the request's metadata
// under `key` and be valid according to the provided `validator`.
func AuthByToken(key string, validator TokenValidator) Middleware {
	return func(next drpc.Handler) drpc.Handler {
		return authToken{
			mKey: key,
			tVal: validator,
			next: next,
		}
	}
}

type authToken struct {
	mKey string
	tVal TokenValidator
	next drpc.Handler
}

func (md authToken) HandleRPC(stream drpc.Stream, rpc string) (err error) {
	data, ok := drpcmetadata.Get(stream.Context())
	if !ok {
		return errors.New("authentication: missing credentials") // no metadata available
	}
	token, ok := data[md.mKey]
	if !ok {
		return errors.New("authentication: missing credentials") // no token set
	}
	if !md.tVal(token) {
		return errors.New("authentication: invalid credentials") // invalid token
	}
	return md.next.HandleRPC(stream, rpc) // continue
}
