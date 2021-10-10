//go:build !go1.16
// +build !go1.16

package did

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"text/template"
)

func (p *Provider) resolve(id *Identifier) ([]byte, error) {
	var err error

	// Parse template
	if p.tpl == nil {
		p.tpl, err = template.New(p.Method).Parse(p.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	// Build URL
	buf := bytes.NewBuffer(nil)
	if err = p.tpl.Execute(buf, p.data(id)); err != nil {
		return nil, err
	}

	// Submit request
	res, err := http.Get(buf.String())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// Return response
	return ioutil.ReadAll(res.Body)
}
