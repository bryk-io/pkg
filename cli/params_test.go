package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Register a list of parameters to a sample command.
func ExampleSetupCommandParams() {
	sampleCmd := &cobra.Command{}
	parameters := []Param{
		{
			Name:      "name-of-parameter",
			Usage:     "describe the parameter use or intent",
			FlagKey:   "cmd.parameter.name",
			ByDefault: 9090,
		},
		{
			Name:      "bool-flag",
			Usage:     "parameters support several basic types",
			FlagKey:   "cmd.parameter.flag",
			ByDefault: false,
		},
	}
	if err := SetupCommandParams(sampleCmd, parameters, viper.GetViper()); err != nil {
		panic(err)
	}
}
