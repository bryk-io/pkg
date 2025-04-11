package json

import (
	"encoding/json"
	"strconv"
	"strings"

	"go.bryk.io/pkg/errors"
)

// Pointer represents a JSON Pointer as defined by RFC6901.
// https://datatracker.ietf.org/doc/html/rfc6901
type Pointer []string

const (
	separator        = "/"
	escapedSeparator = "~1"
	tilde            = "~"
	escapedTilde     = "~0"
)

// ParseJP decodes a provided JSON Pointer.
// The ABNF syntax of a JSON Pointer is:
//
//	json-pointer    = *( "/" reference-token )
//	reference-token = *( unescaped / escaped )
//	unescaped       = %x00-2E / %x30-7D / %x7F-10FFFF [ %x2F ('/') and %x7E ('~') are excluded ]
//	escaped         = "~" ( "0" / "1" ) [ representing '~' and '/', respectively ]
//
// More information:
// https://datatracker.ietf.org/doc/html/rfc6901#section-3
func ParseJP(str string) (Pointer, error) {
	if len(str) == 0 {
		return Pointer{}, nil
	}

	if str[0] != '/' {
		return nil, errors.Errorf("non-empty pointers must start with '/'")
	}
	str = str[1:]

	rft := strings.Split(str, separator)
	for i, t := range rft {
		rft[i] = unescapeRT(t)
	}
	return rft, nil
}

// Return the unescaped representation of the provided reference token.
func unescapeRT(rt string) string {
	rt = strings.ReplaceAll(rt, escapedSeparator, separator)
	return strings.ReplaceAll(rt, escapedTilde, tilde)
}

// Return the escaped representation of the provided reference token.
func escapeRT(rt string) string {
	rt = strings.ReplaceAll(rt, tilde, escapedTilde)
	return strings.ReplaceAll(rt, separator, escapedSeparator)
}

// Evaluation of each reference token begins by decoding any escaped
// character sequence.  This is performed by first transforming any
// occurrence of the sequence '~1' to '/', and then transforming any
// occurrence of the sequence '~0' to '~'.  By performing the
// substitutions in this order, an implementation avoids the error of
// turning '~01' first into '~1' and then into '/', which would be
// incorrect (the string '~01' correctly becomes '~1' after
// transformation).
func evalRT(rt string, data interface{}) (interface{}, error) {
	switch ch := data.(type) {
	case map[string]interface{}:
		v := ch[rt]
		if v == nil {
			return nil, errors.Errorf("invalid reference token: %s", rt)
		}
		return v, nil
	case []interface{}:
		i, err := strconv.Atoi(rt)
		if err != nil {
			return nil, errors.Errorf("invalid array index: %s", rt)
		}
		if i >= len(ch) {
			return nil, errors.Errorf("index %d exceeds array length of %d", i, len(ch))
		}
		return ch[i], nil
	default:
		return nil, errors.Errorf("invalid reference token: %s", rt)
	}
}

// String implements the stringer interface for Pointer, returning the
// escaped string.
func (p Pointer) String() (str string) {
	for _, tok := range p {
		str += "/" + escapeRT(tok)
	}
	return
}

// Eval evaluates a json pointer against the JSON representation of
// the given `src` data element. Evaluation of a JSON Pointer begins
// with a reference to the root value of a JSON document and completes
// with a reference to some value within the document. Each reference
// token in the JSON Pointer is evaluated sequentially.
func (p Pointer) Eval(src interface{}) (result any, err error) {
	// Get JSON document from original data
	doc, err := json.Marshal(src)
	if err != nil {
		return nil, errors.Errorf("invalid source element: %w", err)
	}
	return p.EvalDoc(doc)
}

// EvalDoc evaluates a json pointer against the provided JSON document.
// Evaluation of a JSON Pointer begins with a reference to the root value
// of a JSON document and completes with a reference to some value within
// the document. Each reference token in the JSON Pointer is evaluated
// sequentially.
func (p Pointer) EvalDoc(doc []byte) (result any, err error) {
	// Get root value
	result = make(map[string]any)
	if err = json.Unmarshal(doc, &result); err != nil {
		return nil, errors.Errorf("invalid source element: %w", err)
	}

	// Iterate reference tokens in the pointer instance
	for _, rt := range p {
		if result, err = evalRT(rt, result); err != nil {
			return nil, err
		}
	}
	return
}
