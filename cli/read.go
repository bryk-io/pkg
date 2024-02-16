package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"

	"go.bryk.io/pkg/errors"
	"golang.org/x/term"
)

// ReadSecure will interactively prompt the user to enter a value. The value
// provided won't be displayed on the screen.
func ReadSecure(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	defer fmt.Println()
	return term.ReadPassword(0)
}

// ReadPipedInput retrieves contents passed-in from standard input up to the
// provided maximum number of bytes.
func ReadPipedInput(maxLength int) ([]byte, error) {
	var input []byte

	// Fail to read stdin
	info, err := os.Stdin.Stat()
	if err != nil {
		return input, errors.Wrap(err, "failed to read stdin")
	}

	// No input passed in
	if info.Mode()&os.ModeCharDevice != 0 {
		return input, errors.New("no piped input")
	}

	// Read input
	reader := bufio.NewReader(os.Stdin)
	for {
		b, err := reader.ReadByte()
		if err != nil && errors.Is(err, io.EOF) {
			break
		}
		input = append(input, b)
		if len(input) == maxLength {
			break
		}
	}

	// Return provided input
	return input, nil
}

// SignalsHandler returns a properly configured OS signals handler channel.
func SignalsHandler(list []os.Signal) chan os.Signal {
	signalsCh := make(chan os.Signal, 1)
	signal.Reset(list...)
	signal.Notify(signalsCh, list...)
	return signalsCh
}
