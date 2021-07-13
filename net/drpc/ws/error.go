package ws

import (
	"encoding/json"
	"fmt"

	"storj.io/drpc/drpcerr"
)

type proxyErr struct {
	status int
	err    error
}

func newProxyErr(status int, format string, args ...interface{}) *proxyErr {
	return wrapErr(status, fmt.Errorf(format, args...))
}

func wrapErr(status int, err error) *proxyErr {
	return &proxyErr{status: status, err: err}
}

func (pe *proxyErr) Error() string {
	return pe.err.Error()
}

func (pe *proxyErr) Cause() error {
	return pe.err
}

func (pe *proxyErr) Unwrap() error {
	return pe.err
}

func (pe *proxyErr) JSON() string {
	code := "unknown"
	if dc := drpcerr.Code(pe.err); dc != 0 {
		code = fmt.Sprintf("drpcerr(%d)", dc)
	}
	data, _ := json.Marshal(map[string]interface{}{
		"code": code,
		"msg":  pe.err.Error(),
	})
	return string(data)
}
