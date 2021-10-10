package did

import (
	"encoding/json"
)

// MarshalJSON provides the custom JSON encoding implementation for an identifier instance.
func (d *Identifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON provides custom JSON decoding implementation for an identifier instance.
func (d *Identifier) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	c, err := Parse(s)
	if err != nil {
		return err
	}
	*d = *c
	return nil
}
