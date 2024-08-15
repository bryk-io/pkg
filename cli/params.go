package cli

import (
	"github.com/spf13/cobra"
	"go.bryk.io/pkg/errors"
)

// Param represents an individual CLI parameter.
type Param struct {
	// Name of the parameter, will be displayed to the user when inspecting the
	// help information for the command.
	Name string

	// Brief and clear description of the parameter usage or intent, will be
	// displayed to the user when inspecting the help information for the command.
	Usage string

	// Internal code for the parameter. This should match the structure of a
	// configuration file when used and can be useful to add 'namespaces' for
	// configuration settings. This key should be used when reading configuration
	// options using viper. For example:
	//   root.child.parameter
	FlagKey string

	// Default value to use for the parameter, the type of the default value will
	// determine the expected type for the parameter. Supported types are:
	// int, int32, int64 uint32, uint64, string, bool, []string
	ByDefault interface{}

	// If provided the parameter can be provided using a shorthand letter that can
	// be used after a single dash. Must be unique.
	Short string

	// Parameters are optional by default. If instead you wish your command to report
	// an error when a parameter has not been set, mark it as required.
	Required bool
}

// SetupCommandParams will properly configure the command with the provided parameter list.
func SetupCommandParams(c *cobra.Command, params []Param) error {
	for _, p := range params {
		var err error
		switch v := p.ByDefault.(type) {
		case int:
			err = loadInt(v, c, p)
		case int32:
			err = loadInt32(v, c, p)
		case int64:
			err = loadInt64(v, c, p)
		case float32:
			err = loadFloat32(v, c, p)
		case float64:
			err = loadFloat64(v, c, p)
		case uint32:
			err = loadUnit32(v, c, p)
		case uint64:
			err = loadUnit64(v, c, p)
		case string:
			err = loadString(v, c, p)
		case bool:
			err = loadBool(v, c, p)
		case []string:
			err = loadStringSlice(v, c, p)
		}
		if err != nil {
			return err
		}
		if p.Required {
			if err := errors.WithStack(c.MarkFlagRequired(p.Name)); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadInt(v int, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(int)
	if !ok {
		return errors.New("failed to parse int value")
	}
	if p.Short != "" {
		c.Flags().IntVarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().IntVar(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadInt32(v int32, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(int32)
	if !ok {
		return errors.New("failed to parse int32 value")
	}
	if p.Short != "" {
		c.Flags().Int32VarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().Int32Var(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadInt64(v int64, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(int64)
	if !ok {
		return errors.New("failed to parse int64 value")
	}
	if p.Short != "" {
		c.Flags().Int64VarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().Int64Var(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadFloat32(v float32, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(float32)
	if !ok {
		return errors.New("failed to parse float32 value")
	}
	if p.Short != "" {
		c.Flags().Float32VarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().Float32Var(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadFloat64(v float64, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(float64)
	if !ok {
		return errors.New("failed to parse float64 value")
	}
	if p.Short != "" {
		c.Flags().Float64VarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().Float64Var(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadUnit32(v uint32, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(uint32)
	if !ok {
		return errors.New("failed to parse uint32 value")
	}
	if p.Short != "" {
		c.Flags().Uint32VarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().Uint32Var(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadUnit64(v uint64, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(uint64)
	if !ok {
		return errors.New("failed to parse uint64 value")
	}
	if p.Short != "" {
		c.Flags().Uint64VarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().Uint64Var(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadString(v string, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(string)
	if !ok {
		return errors.New("failed to parse string value")
	}
	if p.Short != "" {
		c.Flags().StringVarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().StringVar(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadBool(v bool, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.(bool)
	if !ok {
		return errors.New("failed to parse bool value")
	}
	if p.Short != "" {
		c.Flags().BoolVarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().BoolVar(&h, p.Name, v, p.Usage)
	}
	return nil
}

func loadStringSlice(v []string, c *cobra.Command, p Param) error {
	h, ok := p.ByDefault.([]string)
	if !ok {
		return errors.New("failed to parse string slice")
	}
	if p.Short != "" {
		c.Flags().StringSliceVarP(&h, p.Name, p.Short, v, p.Usage)
	} else {
		c.Flags().StringSliceVar(&h, p.Name, v, p.Usage)
	}
	return nil
}
