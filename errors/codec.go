package errors

// Codec implementations provide a pluggable way to manage error across
// service boundaries; for example when transmitting error messages through
// a network.
type Codec interface {
	// Encodes an error instance and produce a report.
	Marshal(err error) ([]byte, error)

	// Decoded an error report and restore an error instance.
	// If this operation fails for whatever reason `ok` should
	// be `false`.
	Unmarshal(src []byte) (ok bool, err error)
}

// Report an error instance by generating a portable/transmissible
// representation of it using the provided codec.
func Report(err error, cc Codec) ([]byte, error) {
	return cc.Marshal(err)
}
