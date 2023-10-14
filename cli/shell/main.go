package shell

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/chzyer/readline"
	"go.bryk.io/pkg/errors"
)

// Instance defines the main interface for an interactive shell.
type Instance struct {
	// Settings
	prompt       string   // CLI prompt value used by the shell
	historyFile  string   // Store a history of tasks run in the shell
	historyLimit int      // Maximum number of items to store in the shell history, set to 0 to disable it
	startMessage string   // Message printed just after the shell is started
	exitMessage  string   // Message printed just before the shell is closed
	exitCommands []string // Reserved keywords to let the user close a running shell instance
	helpMessage  string   // Message printed along the command list for the user
	helpCommands []string // Reserved keywords to present a list of available top commands to the user

	// Hooks
	startHook Hook // Custom functionality to run before the shell instance is started
	stopHook  Hook // Custom functionality to run before the shell instance is closed
	resetHook Hook // Custom functionality to run just after the shell state is reset

	// Internal elements
	mu       sync.Mutex
	rl       *readline.Instance
	cfg      *readline.Config
	commands []*Command
}

// New ready-to-use interactive shell instance based on the provided configuration options.
func New(options ...Option) (*Instance, error) {
	// Starts a new instance with a minimal set of sane configuration parameters
	sh := &Instance{
		prompt:       "\033[35m»\033[0m ",
		exitCommands: []string{"exit", "bye"},
		helpCommands: []string{"?", "help"},
		historyLimit: 0,
	}

	// Apply provided settings
	if err := sh.setup(options...); err != nil {
		return nil, errors.Wrap(err, "setup error")
	}

	// Prepare internals
	conf := &readline.Config{
		Prompt:       sh.prompt,
		HistoryFile:  sh.historyFile,
		HistoryLimit: sh.historyLimit,
	}
	rl, err := readline.NewEx(conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	sh.mu.Lock()
	sh.rl = rl
	sh.cfg = conf
	sh.mu.Unlock()
	sh.updateAutoComplete()
	return sh, nil
}

// ReadSecret allows the user to securely and interactively provide a sensitive value.
// The entered data won't be displayed.
func (sh *Instance) ReadSecret(prompt string) ([]byte, error) {
	sh.mu.Lock()
	old := sh.prompt
	sh.mu.Unlock()
	defer sh.SetPrompt(old)
	r, err := sh.rl.ReadPassword(fmt.Sprintf("%s: ", prompt))
	return r, errors.WithStack(err)
}

// ReadString allows the user to interactively provide a value.
func (sh *Instance) ReadString(prompt string) (string, error) {
	sh.mu.Lock()
	old := sh.prompt
	sh.mu.Unlock()
	sh.SetPrompt(fmt.Sprintf("%s: ", prompt))
	defer sh.SetPrompt(old)
	return sh.rl.Operation.String()
}

// ReadSlice is an utility method allowing the user to interactively provide a list of values.
// Each entry in the provided list will be used as a prompt value. The returned list of values
// will be ordered as entered. The processing will terminate on the first error encountered.
func (sh *Instance) ReadSlice(params []string) (list []string, err error) {
	var entry string
	sh.mu.Lock()
	old := sh.prompt
	sh.mu.Unlock()
	defer sh.SetPrompt(old)
	for _, p := range params {
		sh.SetPrompt(fmt.Sprintf("%s: ", p))
		entry, err = sh.rl.Operation.String()
		if err != nil {
			return list, errors.WithStack(err)
		}
		list = append(list, entry)
	}
	return
}

// SetPrompt update the command prompt used by the shell.
func (sh *Instance) SetPrompt(prompt string) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.prompt = prompt
	sh.rl.SetPrompt(prompt)
}

// SetStartHook update the start-hook currently registered on the shell.
func (sh *Instance) SetStartHook(hk Hook) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.startHook = hk
}

// SetStopHook update the stop-hook currently registered on the shell.
func (sh *Instance) SetStopHook(hk Hook) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.stopHook = hk
}

// SetResetHook update the reset-hook currently registered on the shell.
func (sh *Instance) SetResetHook(hk Hook) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.resetHook = hk
}

// Print will add content to the shell's main output.
func (sh *Instance) Print(line string) {
	fmt.Println(line)
}

// AddCommand will register a command with the shell instance and update the autocomplete
// mechanism accordingly.
func (sh *Instance) AddCommand(cmd *Command) {
	sh.mu.Lock()
	sh.commands = append(sh.commands, cmd)
	sh.mu.Unlock()
	sh.updateAutoComplete()
}

// ResetCommands remove all available commands in the shell, useful only when reusing
// an instance with a new command set.
func (sh *Instance) ResetCommands() {
	sh.mu.Lock()
	sh.commands = []*Command{}
	sh.mu.Unlock()
	completer := readline.NewPrefixCompleter()
	conf := sh.cfg
	conf.AutoComplete = completer
	sh.rl.SetConfig(conf)
}

// Reset the shell internal state.
func (sh *Instance) Reset() {
	sh.rl.Clean()
	sh.rl.Refresh()
	if sh.resetHook != nil {
		sh.resetHook()
	}
}

// Clear the console screen.
func (sh *Instance) Clear() {
	// Intentionally ignore the possible error returned when clearing the screen
	_, _ = readline.ClearScreen(sh.rl)
}

// Start the interactive shell processing.
func (sh *Instance) Start() {
	defer func() {
		_ = sh.close()
	}()
	if sh.startHook != nil {
		sh.startHook()
	}

	sh.mu.Lock()
	exitCommands := sh.exitCommands
	sh.mu.Unlock()

	// Show start message
	if sh.startMessage != "" {
		fmt.Println(sh.startMessage)
	}

	// Show help instructions
	fmt.Printf("For help use: %s\n", strings.Join(sh.helpCommands, ", "))

	// Start read-line processing
	for {
		line, err := sh.rl.Readline()
		// exit on interrupt or EOF
		if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
			return
		}

		// check if line is empty
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Look for exit commands
		for _, ec := range exitCommands {
			if line == ec {
				return
			}
		}

		// Look for help commands
		if sh.shouldShowHelp(line) {
			sh.help(sh.commands)
			continue
		}

		// Execute the requested command
		if ok := sh.match(line); !ok {
			fmt.Println("unrecognized command: ", line)
		}
	}
}

// Apply provided configuration options.
func (sh *Instance) setup(options ...Option) error {
	for _, opt := range options {
		if err := opt(sh); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// Finish shell session.
func (sh *Instance) close() error {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if sh.stopHook != nil {
		sh.stopHook()
	}
	if sh.exitMessage != "" {
		fmt.Println(sh.exitMessage)
	}
	return errors.WithStack(sh.rl.Close())
}

// Rebuild autocomplete information based on the currently registered commands.
func (sh *Instance) updateAutoComplete() {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Nothing to do!
	if len(sh.commands) == 0 {
		return
	}

	// Sort command entries by name
	sort.Slice(sh.commands, func(i, j int) bool {
		return sh.commands[i].Name < sh.commands[j].Name
	})

	// Add commands to the auto-completer
	items := make([]readline.PrefixCompleterInterface, len(sh.commands))
	for i, c := range sh.commands {
		items[i] = c.getPCI()
	}
	if len(items) > 0 {
		completer := readline.NewPrefixCompleter(items...)
		conf := sh.cfg
		conf.AutoComplete = completer
		sh.rl.SetConfig(conf)
	}
}

// Inspect a user line and determine if help is being requested.
func (sh *Instance) shouldShowHelp(line string) (ok bool) {
	for _, ec := range sh.helpCommands {
		if line == ec {
			ok = true
			return
		}
	}
	return
}

// Show help information for a command branch.
func (sh *Instance) help(cmd []*Command) {
	if len(cmd) > 0 {
		// Sort command entries by name
		sort.Slice(cmd, func(i, j int) bool {
			return cmd[i].Name < cmd[j].Name
		})

		// Get proper command template
		padding := 0
		for _, c := range cmd {
			if len(c.Name) > padding {
				padding = len(c.Name)
			}
		}
		tpl := fmt.Sprintf("  %%-%ds %%s\n", padding+4)

		// Print commands list
		fmt.Println("Available commands: ")
		for _, c := range cmd {
			fmt.Printf(tpl, c.Name, c.Description)
		}
		fmt.Println("")
	}
	if sh.helpMessage != "" {
		fmt.Println(sh.helpMessage)
	}
	fmt.Printf("To close the session use: %s\n", strings.Join(sh.exitCommands, ", "))
}

// Match an incoming user line with a proper command to execute.
func (sh *Instance) match(line string) (ok bool) {
	// Iterate command branches looking for a proper match
	for _, c := range sh.commands {
		if m, res := c.match(line, sh); m {
			if res != "" {
				sh.Print(res)
			}
			ok = true
			break
		}
	}
	return
}
