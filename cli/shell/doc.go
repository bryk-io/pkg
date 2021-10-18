/*
Package shell provides an interactive client for CLI-based applications.

The main point of interaction is a shell 'Instance'. Specific functionality
is registered with the instance in the form of commands. Commands will be
executed when the user invokes them as part of an active interactive session.

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
*/
package shell
