package cli

import (
	"log"
	"os"
	"syscall"
)

func ExampleReadSecure() {
	// Interactively prompt the user to enter a value. The value provided won't be
	// displayed on the screen.
	password, err := ReadSecure("Enter your password: ")
	if err != nil {
		// Handle error
	}
	log.Printf("you entered: %s", password)
}

func ExampleReadPipedInput() {
	// Read a maximum of 32 bytes from standard input
	input, err := ReadPipedInput(32)
	if len(input) > 0 && err != nil {
		// Handle error
	}
	log.Printf("data received: %s", input)
}

func ExampleSignalsHandler() {
	// Register the signals to look for and wait for one
	s := <-SignalsHandler([]os.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	})
	log.Printf("signal received: %s", s)
}
