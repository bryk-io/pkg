package did

// Param represents a parsed DID param, which contains a name and value.
// A generic param is defined as a param name and value separated by a colon:
//
//	generic-param-name:param-value
//
// https://w3c.github.io/did-core/#generic-did-parameter-names
//
// A param may also be method specific, which requires the method name to
// prefix the param name separated by a colon:
//
//	method-name:param-name
//	param = param-name [ "=" param-value ]
//
// https://w3c.github.io/did-core/#method-specific-did-parameter-names
type Param struct {
	// param-name = 1*param-char
	// Name may include a method name and param name separated by a colon
	Name string
	// param-value = *param-char
	Value string
}

// String encodes a Param struct into a valid Param string.
// Name is required by the grammar. Value is optional.
func (p *Param) String() string {
	if p.Name == "" {
		return ""
	}

	if 0 < len(p.Value) {
		return p.Name + "=" + p.Value
	}

	return p.Name
}
