package amqp

import (
	"fmt"
)

func getName(prefix string) string {
	seed := make([]byte, 6)
	return fmt.Sprintf("%s-%x", prefix, seed)
}
