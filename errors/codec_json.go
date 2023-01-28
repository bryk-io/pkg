package errors

import "encoding/json"

// CodecJSON encodes error data as JSON documents. If `pretty`
// is set to `true` the output will be indented for readability.
func CodecJSON(pretty bool) Codec {
	return &jsonCodec{pretty: pretty}
}

type jsonCodec struct {
	pretty bool
}

func (c *jsonCodec) Marshal(err error) ([]byte, error) {
	data := map[string]interface{}{
		"error": err.Error(),
	}
	var oe *Error
	if As(err, &oe) {
		data["stamp"] = oe.Stamp()
		data["trace"] = oe.PortableTrace()
		if hints := oe.Hints(); len(hints) > 0 {
			data["hints"] = hints
		}
		if tags := oe.Tags(); len(tags) > 0 {
			data["tags"] = tags
		}
		if ev := oe.Events(); ev != nil {
			data["events"] = ev
		}
	}
	if c.pretty {
		return json.MarshalIndent(data, "", "  ")
	}
	return json.Marshal(data)
}
