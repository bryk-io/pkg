package shell

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
)

func TestInstance(t *testing.T) {
	// Skip when running on CI to prevent exhausting resources
	// on the runner.
	if os.Getenv("CI") != "" || os.Getenv("CI_WORKSPACE") != "" {
		t.Skip("CI environment")
		return
	}

	assert := tdd.New(t)

	// Configure shell instance
	options := []Option{
		WithPrompt("sample shell > "),
		WithStartMessage("this is just a sample shell"),
		WithExitMessage("exiting sample shell..."),
		WithExitCommands([]string{"quit"}),
		WithHelpMessage("this is a sample help message"),
		WithHelpCommands([]string{"??"}),
		WithHistoryFile("shell_history"),
		WithHistoryLimit(100),
		WithStartHook(func() {
			log.Println("about to start")
		}),
		WithStopHook(func() {
			log.Println("about to quit")
		}),
		WithResetHook(func() {
			log.Println("applying reset")
		}),
	}
	sh, err := New(options...)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	// Add sample commands
	sh.AddCommand(&Command{
		Name:        "tree",
		Description: "Present a tree of commands with sub-commands",
		SubCommands: []*Command{
			{
				Name: "foo",
				Run: func(arg string) string {
					return "foo"
				},
				SubCommands: []*Command{
					{
						Name: "child-1",
						Run: func(arg string) string {
							return "child-1"
						},
					},
				},
			},
		},
	})
	sh.AddCommand(&Command{
		Name:        "echo",
		Description: "Print back any passed arguments",
		Usage:       "Send anything to print",
		Run: func(args string) string {
			return fmt.Sprintf("you said: %s", args)
		},
	})
	sh.AddCommand(&Command{
		Name:        "now",
		Description: "Current time",
		Run: func(args string) string {
			return fmt.Sprintf("time: %s", time.Now())
		},
	})
	sh.AddCommand(&Command{
		Name:        "clear",
		Description: "Clear the contents from the shell",
		Run: func(args string) string {
			sh.Clear()
			return ""
		},
	})
	sh.AddCommand(&Command{
		Name:        "input-simple",
		Description: "Ask for a simple input entry",
		Run: func(args string) string {
			val, err := sh.ReadString("enter something")
			if err != nil {
				sh.Print(fmt.Sprintf("ERROR: %s", err))
			}
			return fmt.Sprintf("you enter: %s", val)
		},
	})
	sh.AddCommand(&Command{
		Name:        "input-multiple",
		Description: "Ask for multiple input entries",
		Run: func(args string) string {
			ask := []string{"foo", "bar", "baz"}
			list, err := sh.ReadSlice(ask)
			if err != nil {
				sh.Print(fmt.Sprintf("ERROR: %s", err))
			}
			for i, v := range list {
				sh.Print(fmt.Sprintf("for '%s' you enter: %s", ask[i], v))
			}
			return ""
		},
	})
	sh.AddCommand(&Command{
		Name:        "input-secure",
		Description: "Ask for a secret input entry",
		Run: func(args string) string {
			val, err := sh.ReadSecret("enter a secret value")
			if err != nil {
				sh.Print(fmt.Sprintf("ERROR: %s", err))
			}
			return fmt.Sprintf("you enter: %s", val)
		},
	})

	// Start shell instance on the background
	go sh.Start()
	defer func() {
		_ = os.Remove("shell_history")
	}()

	_, err = sh.ReadString("simple variable")
	assert.Equal(io.EOF, err, "read string error")
	assert.Nil(sh.close(), "shell close")
}

// Start a sample shell instance using most of the available options.
func ExampleNew() {
	// Configure shell instance
	options := []Option{
		WithPrompt("sample shell > "),
		WithStartMessage("this is just a sample shell"),
		WithExitMessage("exiting sample shell..."),
		WithExitCommands([]string{"quit"}),
		WithHelpMessage("this is a sample help message"),
		WithHelpCommands([]string{"??"}),
		WithHistoryFile("shell_history"),
		WithHistoryLimit(100),
		WithStartHook(func() {
			log.Println("about to start")
		}),
		WithStopHook(func() {
			log.Println("about to quit")
		}),
		WithResetHook(func() {
			log.Println("applying reset")
		}),
	}
	sh, err := New(options...)
	if err != nil {
		panic(err)
	}
	sh.Start()
}

// Start new shell and add a sample command.
func ExampleInstance_AddCommand() {
	// Create a new shell instance with basic configuration options
	options := []Option{
		WithPrompt("sample shell > "),
		WithStartMessage("this is just a sample shell"),
		WithExitMessage("exiting sample shell..."),
	}
	sh, err := New(options...)
	if err != nil {
		panic(err)
	}

	// Register commands
	sh.AddCommand(&Command{
		Name:        "sample-command",
		Description: "this is just a sample command",
		Usage:       "sample",
		Run: func(_ string) string {
			// Read secure user input
			pass, err := sh.ReadSecret("enter password:")
			if err != nil {
				return fmt.Sprintf("an error occurred: %s", err)
			}
			// Return final result, it will be displayed back to the user
			return fmt.Sprintf("the password entered is: %s", pass)
		},
	})

	// Start interactive session
	sh.Start()
}
