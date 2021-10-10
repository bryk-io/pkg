//go:build go1.16
// +build go1.16

package did

import (
	"bytes"
	"context"
	"io"
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
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, buf.String(), nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// Return response
	return io.ReadAll(res.Body)
}
