package amqp

import (
	"crypto/rand"
	"fmt"
)

func getName(prefix string) string {
	seed := make([]byte, 4)
	_, _ = rand.Read(seed)
	return fmt.Sprintf("%s-%x", prefix, seed)
}
