package errors

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CodecJSON encodes error data as JSON documents. If `pretty`
// is set to `true` the output will be indented for readability.
func CodecJSON(pretty bool) Codec {
	return &jsonCodec{pretty: pretty}
}

type errReport struct {
	Msg    string                 `json:"error,omitempty"`
	Stamp  int64                  `json:"stamp,omitempty"`
	Trace  []StackFrame           `json:"trace,omitempty"`
	Hints  []string               `json:"hints,omitempty"`
	Tags   map[string]interface{} `json:"tags,omitempty"`
	Events []Event                `json:"events,omitempty"`
}

type jsonCodec struct {
	pretty bool
}

func (c *jsonCodec) Marshal(err error) ([]byte, error) {
	rec := new(errReport)
	rec.Msg = err.Error()
	var oe *Error
	if As(err, &oe) {
		rec.Stamp = oe.Stamp()
		rec.Trace = oe.PortableTrace() // oe.StackTrace()
		rec.Hints = oe.Hints()
		rec.Tags = oe.Tags()
		rec.Events = oe.Events()
	}
	if c.pretty {
		return json.MarshalIndent(rec, "", "  ")
	}
	return json.Marshal(rec)
}

func (c *jsonCodec) Unmarshal(src []byte) (bool, error) {
	// validate error report
	rep := new(errReport)
	if err := json.Unmarshal(src, rep); err != nil {
		return false, nil
	}

	// restore recovered error details
	rec := new(Error)
	rec.ts = rep.Stamp
	rec.frames = rep.Trace
	rec.hints = rep.Hints
	rec.tags = rep.Tags
	rec.events = rep.Events

	// parse error message
	msg := strings.Split(rep.Msg, ":")
	if len(msg) > 1 {
		rec.prefix = msg[0]
		rec.err = fmt.Errorf("%s", strings.Join(msg[1:], ": "))
	} else {
		rec.err = fmt.Errorf("%s", rep.Msg)
	}
	return true, rec
}
