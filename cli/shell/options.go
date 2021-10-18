package shell

// Option provides a functional method to adjust the settings on a shell instance.
type Option func(*Instance) error

// Hook provides a mechanism to extend the shell functionality during several
// moments of its lifecycle.
type Hook func()

// WithPrompt set the CLI prompt value used by the shell.
func WithPrompt(p string) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.prompt = p
		return nil
	}
}

// WithHistoryFile adjust the location to store a log of tasks executed in the shell.
func WithHistoryFile(hf string) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.historyFile = hf
		return nil
	}
}

// WithHistoryLimit set the maximum number of items to store in the shell history,
// set to 0 to disable it.
func WithHistoryLimit(limit int) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.historyLimit = limit
		return nil
	}
}

// WithStartMessage set a message to be printed just after the shell is started.
func WithStartMessage(msg string) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.startMessage = msg
		return nil
	}
}

// WithExitMessage set a message to be printed just before the shell is closed.
func WithExitMessage(msg string) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.exitMessage = msg
		return nil
	}
}

// WithExitCommands provides a list of reserved keywords to let the user close a
// running shell instance.
func WithExitCommands(list []string) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.exitCommands = list
		return nil
	}
}

// WithHelpMessage set a message to be printed along the command list for the user.
func WithHelpMessage(msg string) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.helpMessage = msg
		return nil
	}
}

// WithHelpCommands provides a list of reserved keywords to present a list of available
// top commands to the user.
func WithHelpCommands(list []string) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.helpCommands = list
		return nil
	}
}

// WithStartHook set a custom behavior to run before the shell instance is started.
func WithStartHook(hook Hook) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.startHook = hook
		return nil
	}
}

// WithStopHook set a custom behavior to run before the shell instance is closed.
func WithStopHook(hook Hook) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.stopHook = hook
		return nil
	}
}

// WithResetHook set a custom behavior to run just after the shell state is reset.
func WithResetHook(hook Hook) Option {
	return func(sh *Instance) error {
		sh.mu.Lock()
		defer sh.mu.Unlock()
		sh.resetHook = hook
		return nil
	}
}
