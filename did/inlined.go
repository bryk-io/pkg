package did

import (
	"fmt"
)

// Calls to all functions below this point should be inlined by the go compiler
// See output of `go build -gcflags -m` to confirm

// Returns true if a byte is not allowed in a ID from the grammar:
//   idchar = ALPHA / DIGIT / "." / "-"
func isNotValidIDChar(char byte) bool {
	return isNotAlpha(char) && isNotDigit(char) && char != '.' && char != '-'
}

// isNotValidParamChar returns true if a byte is not allowed in a param-name
// or param-value from the grammar:
//   idchar = ALPHA / DIGIT / "." / "-" / "_" / ":"
func isNotValidParamChar(char byte) bool {
	return isNotAlpha(char) && isNotDigit(char) &&
		char != '.' && char != '-' && char != '_' && char != ':'
}

// isNotValidQueryOrFragmentChar returns true if a byte is not allowed in a Fragment
// from the grammar:
//   did-fragment = *( pchar / "/" / "?" )
//   pchar        = unreserved / pct-encoded / sub-delims / ":" / "@"
func isNotValidQueryOrFragmentChar(char byte) bool {
	return isNotValidPathChar(char) && char != '/' && char != '?'
}

// Returns true if a byte is not allowed in Path
//   did-path    = segment-nz *( "/" segment )
//   segment     = *pchar
//   segment-nz  = 1*pchar
//   pchar       = unreserved / pct-encoded / sub-delims / ":" / "@"
func isNotValidPathChar(char byte) bool {
	return isNotUnreservedOrSubdelim(char) && char != ':' && char != '@'
}

// Returns true if a byte is not unreserved or sub-delims from the grammar:
//   unreserved = ALPHA / DIGIT / "-" / "." / "_" / "~"
//   sub-delims = "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
// https://tools.ietf.org/html/rfc3986#appendix-A
func isNotUnreservedOrSubdelim(char byte) bool {
	switch char {
	case '-', '.', '_', '~', '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=':
		return false
	default:
		if isNotAlpha(char) && isNotDigit(char) {
			return true
		}
		return false
	}
}

// Returns true if a byte is not a digit between 0-9 or A-F or a-f in US-ASCII
// https://tools.ietf.org/html/rfc5234#appendix-B.1
func isNotHexDigit(char byte) bool {
	// '\x41' is A, '\x46' is F
	// '\x61' is a, '\x66' is f
	// isNotDigit(char) && (char < '\x41' || char > '\x46') && (char < '\x61' || char > '\x66')
	return isNotDigit(char) && (char < '\x41' || char > '\x46') && (char < '\x61' || char > '\x66')
}

// Returns true if a byte is not a digit between 0-9 in US-ASCII
// https://tools.ietf.org/html/rfc5234#appendix-B.1
func isNotDigit(char byte) bool {
	// '\x30' is digit 0, '\x39' is digit 9
	return char < '\x30' || char > '\x39'
}

// Returns true if a byte is not a big letter between A-Z or small letter between a-z
// https://tools.ietf.org/html/rfc5234#appendix-B.1
func isNotAlpha(char byte) bool {
	return isNotSmallLetter(char) && isNotBigLetter(char)
}

// Returns true if a byte is not a big letter between A-Z in US-ASCII
// https://tools.ietf.org/html/rfc5234#appendix-B.1
func isNotBigLetter(char byte) bool {
	// '\x41' is big letter A, '\x5A' small letter Z
	return char < '\x41' || char > '\x5A'
}

// Returns true if a byte is not a small letter between a-z in US-ASCII
// https://tools.ietf.org/html/rfc5234#appendix-B.1
func isNotSmallLetter(char byte) bool {
	// '\x61' is small letter a, '\x7A' small letter z
	return char < '\x61' || char > '\x7A'
}

// Wrap an error message. If 'err' is nil, this method return nil as well.
func wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
