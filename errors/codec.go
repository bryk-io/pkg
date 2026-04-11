package errors

// Codec implementations provide a pluggable way to manage error across
// service boundaries; for example when transmitting error messages through
// a network.
type Codec interface {
	// Encodes an error instance and produce a report.
	Marshal(err error) ([]byte, error)

	// Decodes an error report and restores an error instance.
	// Returns the recovered error, or nil if the operation fails
	// (e.g., invalid data format, corruption, etc.).
	Unmarshal(src []byte) error
}

// Report an error instance by generating a portable/transmissible
// representation of it using the provided codec.
func Report(err error, cc Codec) ([]byte, error) {
	return cc.Marshal(err)
}
