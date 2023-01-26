package errors

// Codec implementations provide a pluggable way to manage error across
// service boundaries; for example when transmitting error messages through
// a network.
type Codec interface {
	Marshal(err error) ([]byte, error)
}

// Report an error instance by generating a portable/transmissible
// representation of it using the provided codec.
func Report(err error, cc Codec) ([]byte, error) {
	return cc.Marshal(err)
}
