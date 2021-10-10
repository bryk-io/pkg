package did

import (
	"errors"
	"text/template"
)

// Resolve the DID document (or the provider's response) for the provided
// identifier instance.
func Resolve(id string, providers []*Provider) ([]byte, error) {
	// Validate id
	r, err := Parse(id)
	if err != nil {
		return nil, err
	}

	// Select provider
	var p *Provider
	for _, p = range providers {
		if p.Method == r.Method() {
			break
		}
	}
	if p == nil {
		return nil, errors.New("not supported method")
	}

	// Return result
	return p.resolve(r)
}

// Provider represents an external system able to return DID Documents
// on demand.
type Provider struct {
	// Method value expected on the identifier instance.
	Method string

	// Network location to retrieve DID documents from. The value can
	// be a template with support for the following variables: DID, Method
	// and Subject. For example:
	// https://did.baidu.com/v1/did/resolve/{{.DID}}
	Endpoint string

	// Protocol used to communicate with the endpoint. Currently HTTP(S)
	// is supported by submitting GET requests.
	Protocol string

	// Compiled endpoint template
	tpl *template.Template
}

func (p *Provider) data(id *Identifier) map[string]string {
	return map[string]string{
		"DID":     id.String(),
		"Method":  id.Method(),
		"Subject": id.Subject(),
	}
}
