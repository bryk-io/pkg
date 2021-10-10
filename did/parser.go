package did

import (
	"fmt"
	"strings"
)

// Parse a provided input string into a valid identifier instance.
func Parse(input string) (*Identifier, error) {
	// Initialize parser steps
	p := &parser{input: input, out: &Identifier{
		data: &identifierData{},
	}}

	// Execute parser state machine
	parserState := p.checkLength
	for parserState != nil {
		parserState = parserState()
	}

	// If one of the steps added an err to the parser state, exit. Return nil and the error.
	if p.err != nil {
		return nil, wrap(p.err, "invalid DID")
	}

	// join IDStrings with : to make up ID
	p.out.data.ID = strings.Join(p.out.data.IDStrings[:], ":")

	// join PathSegments with / to make up Path
	p.out.data.Path = strings.Join(p.out.data.PathSegments[:], "/")

	return p.out, nil
}

// Original parser version from:
// https://github.com/ockam-network/did
type parser struct {
	input        string
	currentIndex int
	out          *Identifier
	err          error
}

// A step in the parser state machine that returns the next step.
type parserStep func() parserStep

// checkLength is a parserStep that checks if the input length is at least 7
// the grammar requires.
//   `did:` prefix (4 chars)
//   + at least one methodchar (1 char)
//   + `:` (1 char)
//   + at least one idchar (1 char)
// i.e at least 7 chars
// The current specification does not take a position on maximum length of a DID.
// https://w3c-ccg.github.io/did-spec/#upper-limits-on-did-character-length
func (p *parser) checkLength() parserStep {
	if inputLength := len(p.input); inputLength < 7 {
		return p.errorf(inputLength, "input length is less than 7")
	}
	return p.parseScheme
}

// parseScheme is a parserStep that validates that the input begins with 'did:'.
func (p *parser) parseScheme() parserStep {
	currentIndex := 3 // 4 bytes in 'did:', i.e index 3
	if p.input[:currentIndex+1] != prefix {
		return p.errorf(currentIndex, "input does not begin with 'did:' prefix: %s", p.input[:currentIndex+1])
	}
	p.currentIndex = currentIndex
	return p.parseMethod
}

// parseMethod is a parserStep that extracts the DID Method.
// from the grammar:
//   did        = "did:" method ":" specific-idstring
//   method     = 1*methodchar
//   methodchar = %x61-7A / DIGIT ; 61-7A is a-z in US-ASCII
func (p *parser) parseMethod() parserStep {
	input := p.input
	inputLength := len(input)
	currentIndex := p.currentIndex + 1
	startIndex := currentIndex

	// parse method name
	// loop over every byte following the ':' in 'did:' until the second ':'
	// method is the string between the two ':'s
	for {
		if currentIndex == inputLength {
			// we got to the end of the input and didn't find a second ':'
			return p.errorf(currentIndex, "input does not have a second `:` marking end of method name")
		}

		// read the input character at currentIndex
		char := input[currentIndex]

		if char == ':' {
			// we've found the second : in the input that marks the end of the method
			if currentIndex == startIndex {
				// return error is method is empty, ex- did::1234
				return p.errorf(currentIndex, "method is empty")
			}
			break
		}

		// as per the grammar method can only be made of digits 0-9 or small letters a-z
		if isNotDigit(char) && isNotSmallLetter(char) {
			return p.errorf(currentIndex, "character is not a-z OR 0-9")
		}

		// move to the next char
		currentIndex = currentIndex + 1
	}

	// set parser state
	p.currentIndex = currentIndex
	p.out.data.Method = input[startIndex:currentIndex]

	// method is followed by specific-idstring, parse that next
	return p.parseID
}

// parseID is a parserStep that extracts : separated idstrings that are part of
// a specific-idstring and adds them to p.out.IDStrings.
// from the grammar:
//   specific-idstring = idstring *( ":" idstring )
//   idstring          = 1*idchar
//   idchar            = ALPHA / DIGIT / "." / "-"
// p.out.IDStrings is later concatenated by the Parse function before it returns.
func (p *parser) parseID() parserStep {
	input := p.input
	inputLength := len(input)
	currentIndex := p.currentIndex + 1
	startIndex := currentIndex

	var next parserStep
	for {
		if currentIndex == inputLength {
			// we've reached end of input, no next state
			next = nil
			break
		}

		char := input[currentIndex]

		if char == ':' {
			// encountered : input may have another idstring, parse ID again
			next = p.parseID
			break
		}

		if char == ';' {
			// encountered ; input may have a parameter, parse that next
			next = p.parseParamName
			break
		}

		if char == '/' {
			// encountered / input may have a path following specific-idstring, parse that next
			next = p.parsePath
			break
		}

		if char == '?' {
			// encountered ? input may have a query following specific-idstring, parse that next
			next = p.parseQuery
			break
		}

		if char == '#' {
			// encountered # input may have a fragment following specific-idstring, parse that next
			next = p.parseFragment
			break
		}

		// make sure current char is a valid idchar
		// idchar = ALPHA / DIGIT / "." / "-"
		if isNotValidIDChar(char) {
			return p.errorf(currentIndex, "byte is not ALPHA OR DIGIT OR '.' OR '-'")
		}

		// move to the next char
		currentIndex = currentIndex + 1
	}

	if currentIndex == startIndex {
		// idstring length is zero
		// from the grammar:
		//   idstring = 1*idchar
		// return error because idstring is empty, ex- did:a::123:456
		return p.errorf(currentIndex, "idstring must be at least one char long")
	}

	// set parser state
	p.currentIndex = currentIndex
	p.out.data.IDStrings = append(p.out.data.IDStrings, input[startIndex:currentIndex])

	// return the next parser step
	return next
}

// parseParamName is a parserStep that extracts a did-url param-name.
// A Param struct is created for each param name that is encountered.
// from the grammar:
//   param              = param-name [ "=" param-value ]
//   param-name         = 1*param-char
//   param-char         = ALPHA / DIGIT / "." / "-" / "_" / ":" / pct-encoded
func (p *parser) parseParamName() parserStep {
	input := p.input
	startIndex := p.currentIndex + 1
	next := p.paramTransition()
	currentIndex := p.currentIndex

	if currentIndex == startIndex {
		// param-name length is zero
		// from the grammar:
		//   1*param-char
		// return error because param-name is empty, ex- did:a::123:456;param-name
		return p.errorf(currentIndex, "Param name must be at least one char long")
	}

	// Create a new param with the name
	p.out.data.Params = append(p.out.data.Params, Param{Name: input[startIndex:currentIndex], Value: ""})

	// return the next parser step
	return next
}

// parseParamValue is a parserStep that extracts a did-url param-value.
// A parsed Param value requires that a Param was previously created when parsing a param-name.
// from the grammar:
//   param              = param-name [ "=" param-value ]
//   param-value         = 1*param-char
//   param-char         = ALPHA / DIGIT / "." / "-" / "_" / ":" / pct-encoded
func (p *parser) parseParamValue() parserStep {
	input := p.input
	startIndex := p.currentIndex + 1
	next := p.paramTransition()
	currentIndex := p.currentIndex

	// Get the last Param in the DID and append the value
	// values may be empty according to the grammar- *param-char
	p.out.data.Params[len(p.out.data.Params)-1].Value = input[startIndex:currentIndex]

	// return the next parser step
	return next
}

// paramTransition is a parserStep that extracts and transitions a param-name
// or param-value.
// nolint: gocyclo
func (p *parser) paramTransition() parserStep {
	input := p.input
	inputLength := len(input)
	currentIndex := p.currentIndex + 1

	var indexIncrement int
	var next parserStep
	var percentEncoded bool

	for {
		if currentIndex == inputLength {
			// we've reached end of input, no next state
			next = nil
			break
		}

		char := input[currentIndex]

		if char == ';' {
			// encountered : input may have another param, parse paramName again
			next = p.parseParamName
			break
		}

		// Separate steps for name and value?
		if char == '=' {
			// parse param value
			next = p.parseParamValue
			break
		}

		if char == '/' {
			// encountered / input may have a path following current param, parse that next
			next = p.parsePath
			break
		}

		if char == '?' {
			// encountered ? input may have a query following current param, parse that next
			next = p.parseQuery
			break
		}

		if char == '#' {
			// encountered # input may have a fragment following current param, parse that next
			next = p.parseFragment
			break
		}

		if char == '%' {
			// a % must be followed by 2 hex digits
			if (currentIndex+2 >= inputLength) ||
				isNotHexDigit(input[currentIndex+1]) ||
				isNotHexDigit(input[currentIndex+2]) {
				return p.errorf(currentIndex, "%% is not followed by 2 hex digits")
			}
			// if we got here, we're dealing with percent encoded char, jump three chars
			percentEncoded = true
			indexIncrement = 3
		} else {
			// not percent encoded
			percentEncoded = false
			indexIncrement = 1
		}

		// make sure current char is a valid param-char
		// idchar = ALPHA / DIGIT / "." / "-"
		if !percentEncoded && isNotValidParamChar(char) {
			return p.errorf(currentIndex, "character is not allowed in param - %c", char)
		}

		// move to the next char
		currentIndex = currentIndex + indexIncrement
	}

	// set parser state
	p.currentIndex = currentIndex

	return next
}

// parsePath is a parserStep that extracts a DID Path from a DID Reference
// from the grammar:
//   did-path      = segment-nz *( "/" segment )
//   segment       = *pchar
//   segment-nz    = 1*pchar
//   pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
//   unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
//   pct-encoded   = "%" HEXDIG HEXDIG
//   sub-delims    = "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
func (p *parser) parsePath() parserStep {
	input := p.input
	inputLength := len(input)
	currentIndex := p.currentIndex + 1
	startIndex := currentIndex

	var indexIncrement int
	var next parserStep
	var percentEncoded bool

	for {
		if currentIndex == inputLength {
			next = nil
			break
		}

		char := input[currentIndex]

		if char == '/' {
			// encountered / input may have another path segment, try to parse that next
			next = p.parsePath
			break
		}

		if char == '?' {
			// encountered ? input may have a query following path, parse that next
			next = p.parseQuery
			break
		}

		if char == '%' {
			// a % must be followed by 2 hex digits
			if (currentIndex+2 >= inputLength) ||
				isNotHexDigit(input[currentIndex+1]) ||
				isNotHexDigit(input[currentIndex+2]) {
				return p.errorf(currentIndex, "%% is not followed by 2 hex digits")
			}
			// if we got here, we're dealing with percent encoded char, jump three chars
			percentEncoded = true
			indexIncrement = 3
		} else {
			// not percent encoded
			percentEncoded = false
			indexIncrement = 1
		}

		// pchar = unreserved / pct-encoded / sub-delims / ":" / "@"
		if !percentEncoded && isNotValidPathChar(char) {
			return p.errorf(currentIndex, "character is not allowed in path: %v", char)
		}

		// move to the next char
		currentIndex = currentIndex + indexIncrement
	}

	if currentIndex == startIndex && len(p.out.data.PathSegments) == 0 {
		// path segment length is zero
		// first path segment must have at least one character
		// from the grammar
		//   did-path = segment-nz *( "/" segment )
		return p.errorf(currentIndex, "first path segment must have at least one character")
	}

	// update parser state
	p.currentIndex = currentIndex
	p.out.data.PathSegments = append(p.out.data.PathSegments, input[startIndex:currentIndex])
	return next
}

// parseQuery is a parserStep that extracts a DID Query from a DID Reference
// from the grammar:
//   did-query     = *( pchar / "/" / "?" )
//   pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
//   unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
//   pct-encoded   = "%" HEXDIG HEXDIG
//   sub-delims    = "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
func (p *parser) parseQuery() parserStep {
	input := p.input
	inputLength := len(input)
	currentIndex := p.currentIndex + 1
	startIndex := currentIndex

	var indexIncrement int
	var next parserStep
	var percentEncoded bool

	for {
		if currentIndex == inputLength {
			// we've reached the end of input
			// it's ok for query to be empty, so we don't need a check for that
			// did-query     = *( pchar / "/" / "?" )
			break
		}

		char := input[currentIndex]

		if char == '#' {
			// encountered # input may have a fragment following the query, parse that next
			next = p.parseFragment
			break
		}

		if char == '%' {
			// a % must be followed by 2 hex digits
			if (currentIndex+2 >= inputLength) ||
				isNotHexDigit(input[currentIndex+1]) ||
				isNotHexDigit(input[currentIndex+2]) {
				return p.errorf(currentIndex, "%% is not followed by 2 hex digits")
			}
			// if we got here, we're dealing with percent encoded char, jump three chars
			percentEncoded = true
			indexIncrement = 3
		} else {
			// not percent encoded
			percentEncoded = false
			indexIncrement = 1
		}

		// did-query = *( pchar / "/" / "?" )
		// pchar = unreserved / pct-encoded / sub-delims / ":" / "@"
		// isNotValidQueryOrFragmentChar checks for all the valid chars except pct-encoded
		if !percentEncoded && isNotValidQueryOrFragmentChar(char) {
			return p.errorf(currentIndex, "character is not allowed in query - %c", char)
		}

		// move to the next char
		currentIndex = currentIndex + indexIncrement
	}

	// update parser state
	p.currentIndex = currentIndex
	p.out.data.Query = input[startIndex:currentIndex]
	return next
}

// parseFragment is a parserStep that extracts a DID Fragment from a DID Reference
// from the grammar:
//   did-fragment  = *( pchar / "/" / "?" )
//   pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
//   unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
//   pct-encoded   = "%" HEXDIG HEXDIG
//   sub-delims    = "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
func (p *parser) parseFragment() parserStep {
	input := p.input
	inputLength := len(input)
	currentIndex := p.currentIndex + 1
	startIndex := currentIndex

	var indexIncrement int
	var percentEncoded bool

	for {
		if currentIndex == inputLength {
			// we've reached the end of input
			// it's ok for reference to be empty, so we don't need a check for that
			// did-fragment = *( pchar / "/" / "?" )
			break
		}

		char := input[currentIndex]

		if char == '%' {
			// a % must be followed by 2 hex digits
			if (currentIndex+2 >= inputLength) ||
				isNotHexDigit(input[currentIndex+1]) ||
				isNotHexDigit(input[currentIndex+2]) {
				return p.errorf(currentIndex, "%% is not followed by 2 hex digits")
			}
			// if we got here, we're dealing with percent encoded char, jump three chars
			percentEncoded = true
			indexIncrement = 3
		} else {
			// not percent encoded
			percentEncoded = false
			indexIncrement = 1
		}

		// did-fragment = *( pchar / "/" / "?" )
		// pchar = unreserved / pct-encoded / sub-delims / ":" / "@"
		// isNotValidQueryOrFragmentChar checks for all the valid chars except pct-encoded
		if !percentEncoded && isNotValidQueryOrFragmentChar(char) {
			return p.errorf(currentIndex, "character is not allowed in fragment - %c", char)
		}

		// move to the next char
		currentIndex = currentIndex + indexIncrement
	}

	// update parser state
	p.currentIndex = currentIndex
	p.out.data.Fragment = input[startIndex:currentIndex]

	// no more parsing needed after a fragment,
	// cause the state machine to exit by returning nil
	return nil
}

// errorf is a parserStep that returns nil to cause the state machine to exit
// before returning it sets the currentIndex and err field in parser state
// other parser steps use this function to exit the state machine with an error.
func (p *parser) errorf(index int, format string, args ...interface{}) parserStep {
	p.currentIndex = index
	p.err = fmt.Errorf(format, args...)
	return nil
}
