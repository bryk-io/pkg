package viper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	lib "github.com/spf13/viper"
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/errors"
)

// BindFlags will detect flags used to link parameters to the command and
// properly bind each one to the provided viper instance.
func BindFlags(cmd *cobra.Command, params []cli.Param, vp *lib.Viper) error {
	for _, p := range params {
		if err := errors.WithStack(vp.BindPFlag(p.FlagKey, cmd.Flags().Lookup(p.Name))); err != nil {
			return err
		}
	}
	return nil
}

// Config provides a simple interface to manage application settings using Viper.
type Config struct {
	id        string     // main app identifier
	file      string     // config file name (without extension)
	ext       string     // implicit extension for the config file when not present
	locations []string   // additional locations to look for a config file
	vp        *lib.Viper // internal viper instance
}

// ConfigOptions adjust the internal behavior of the configuration handler.
type ConfigOptions struct {
	// Configuration file name (without extension). Defaults to `config`.
	FileName string

	// Configuration file extension. This will be used internally to automatically
	// decode its contents accordingly. Defaults to `yaml`
	FileType string

	// Additional locations to look for the configuration file.
	Locations []string
}

func (co *ConfigOptions) defaults() {
	if co.FileName == "" {
		co.FileName = "config"
	}
	if co.FileType == "" {
		co.FileType = "yaml"
	}
}

// ConfigHandler returns a new configuration management instance configured
// for the provided `app` identifier. Optional locations can be provided to
// specify valid paths to look for a config file.
func ConfigHandler(app string, opts *ConfigOptions) *Config {
	if opts == nil {
		opts = new(ConfigOptions)
	}
	opts.defaults()
	c := &Config{
		id:        app,
		vp:        lib.New(),
		file:      opts.FileName,
		ext:       opts.FileType,
		locations: append([]string{}, opts.Locations...),
	}

	// ENV
	c.vp.SetEnvPrefix(c.id)
	c.vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.vp.AutomaticEnv()

	// Configuration file settings. Default locations:
	// - /etc/{APP}
	// - /home/{APP}
	// - /home/.{APP}
	// - `pwd`
	c.vp.SetConfigName(c.file)
	c.vp.SetConfigType(c.ext)
	c.vp.AddConfigPath(filepath.Join("/etc", c.id))
	if home, err := os.UserHomeDir(); err == nil {
		c.vp.AddConfigPath(filepath.Join(home, c.id))
		c.vp.AddConfigPath(filepath.Join(home, fmt.Sprintf(".%s", c.id)))
	}
	c.vp.AddConfigPath(".")
	for _, loc := range c.locations {
		c.vp.AddConfigPath(loc)
	}
	return c
}

// ReadFile will try to load configuration values from the local filesystem;
// optionally ignore the error produced when no configuration file was found.
func (c *Config) ReadFile(ignoreNotFound bool) error {
	if err := c.vp.ReadInConfig(); err != nil {
		if errors.As(err, new(lib.ConfigFileNotFoundError)) && ignoreNotFound {
			return nil
		}
		return err
	}
	return nil
}

// FileUsed returns the full path of the configuration file used to load the
// settings.
func (c *Config) FileUsed() string {
	return c.vp.ConfigFileUsed()
}

// Read configuration values from the provided `src` element.
func (c *Config) Read(src io.Reader) error {
	return c.vp.ReadConfig(src)
}

// Get the value registered for `key`, if any.
func (c *Config) Get(key string) interface{} {
	return c.vp.Get(key)
}

// Set the provided `key` to `value`.
func (c *Config) Set(key string, value interface{}) {
	c.vp.Set(key, value)
}

// IsSet returns true if a value is available for `key`.
func (c *Config) IsSet(key string) bool {
	return c.vp.IsSet(key)
}

// Unmarshal will load configuration values into `receiver`; which must be a
// pointer. A `key` value can be provided to load a specific subsection of the
// settings available.
func (c *Config) Unmarshal(receiver interface{}, key string) error {
	if key != "" {
		return c.vp.UnmarshalKey(key, receiver)
	}
	return c.vp.Unmarshal(receiver)
}

// Internals expose the private viper instance used by the configuration manager;
// use with care.
func (c *Config) Internals() *lib.Viper {
	return c.vp
}
