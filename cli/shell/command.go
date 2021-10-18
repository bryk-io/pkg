package shell

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

// Command provides a mechanism to add functionality to a shell instance.
type Command struct {
	// Short name for the command.
	Name string

	// Brief but clear description about the command purpose.
	Description string

	// Information about how to use the command.
	Usage string

	// Method to execute when the command is invoked.
	Run func(arg string) string

	// Sub-commands available, if any.
	SubCommands []*Command
}

// Build the proper auto-completer entries. It will handle nested elements as needed.
func (c *Command) getPCI() readline.PrefixCompleterInterface {
	var items []readline.PrefixCompleterInterface
	if c.SubCommands != nil {
		// Sort command entries by name
		sort.Slice(c.SubCommands, func(i, j int) bool {
			return c.SubCommands[i].Name < c.SubCommands[j].Name
		})

		for _, cc := range c.SubCommands {
			items = append(items, cc.getPCI())
		}
	}
	return readline.PcItem(c.Name, items...)
}

// Match an incoming user line with a command branch.
func (c *Command) match(line string, sh *Instance) (bool, string) {
	// Not a match
	if strings.SplitN(line, " ", 2)[0] != c.Name {
		return false, ""
	}

	line = strings.TrimSpace(strings.Replace(line, c.Name, "", 1))
	if c.SubCommands != nil {
		for _, cc := range c.SubCommands {
			if ok, res := cc.match(line, sh); ok {
				return true, res
			}
		}
	}

	// Display help if required
	if sh.shouldShowHelp(line) || c.Run == nil {
		if len(c.SubCommands) > 0 {
			sh.help(c.SubCommands)
			return true, ""
		}
		return true, c.help()
	}

	// Execute command function
	return true, c.Run(line)
}

// Return the help information for a command instance.
func (c *Command) help() string {
	res := c.Description
	if c.Usage != "" {
		res += fmt.Sprintf("\nUsage: %s\n", c.Usage)
	}
	return res
}
