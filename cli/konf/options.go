package konf

import (
	"flag"
	"strings"

	"github.com/spf13/pflag"
)

type settings struct {
	locations []string
	env       bool
	envPrefix string
	tagName   string
	flags     *flag.FlagSet
	pflags    *pflag.FlagSet
}

// Option instances adjust the behavior of a configuration provider.
type Option func(*settings) error

// WithFileLocations adjust the configuration provider to attempt to load a configuration
// file form the local filesystem. The first valid configuration file found will be the one
// used.
func WithFileLocations(locations []string) Option {
	return func(s *settings) error {
		s.locations = locations
		return nil
	}
}

// WithFlags loads configuration values coming from "command-line" flags.
func WithFlags(set *flag.FlagSet) Option {
	return func(s *settings) error {
		s.flags = set
		return nil
	}
}

// WithPflags loads configuration values coming from "command-line" flags using
// the `github.com/spf13/pflag` package.
func WithPflags(set *pflag.FlagSet) Option {
	return func(s *settings) error {
		s.pflags = set
		return nil
	}
}

// WithTagName adjust the tag identifier is used when decoding configuration into structs.
// For example, with the tag name `konf`, it would look for `konf` tags on struct fields.
// If no value is provided a sane default will be used depending on the configuration
// file extension.
func WithTagName(name string) Option {
	return func(s *settings) error {
		s.tagName = name
		return nil
	}
}

// WithEnv adjust the configuration provider to load values from
// ENV variables. If a `prefix` is provided only ENV variables
// with it will be evaluated. The provided `prefix` value will be
// automatically formatted, for example: `myapp` will be evaluated
// as `MYAPP_`.
func WithEnv(prefix string) Option {
	return func(s *settings) error {
		prefix = strings.ToUpper(prefix)
		if !strings.HasSuffix(prefix, "_") {
			prefix = prefix + "_"
		}

		s.env = true
		s.envPrefix = prefix
		return nil
	}
}
