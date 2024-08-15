package konf

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	lib "github.com/nil-go/konf"
	flagP "github.com/nil-go/konf/provider/flag"
	pflagP "github.com/nil-go/konf/provider/pflag"
	"github.com/spf13/pflag"
)

// DefaultLocations returns a common set of paths where to look for a configuration file.
//   - /etc/appName/fileName (on linux and darwin)
//   - /${HOME}/appName/fileName
//   - /${HOME}/.appName/fileName
//   - `pwd`/fileName
func DefaultLocations(appName, fileName string) []string {
	locations := []string{}
	if runtime.GOOS != "windows" {
		locations = append(locations, filepath.Join("/etc", appName, fileName))
	}
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations, filepath.Join(home, appName, fileName))
		locations = append(locations, filepath.Join(home, "."+appName, fileName))
	}
	if cwd, err := os.Getwd(); err != nil {
		locations = append(locations, filepath.Join(cwd, fileName))
	}
	return locations
}

// PflagsLoader checks the config instance to verify if the defined pflags have
// been set by other providers. If not, default pflag values are merged. If
// they exist, pflag values are merged only if explicitly set in the command line.
// The `splitter` value is used to split flag names into nested keys, if not
// provided the default value "." will be used.
func PflagsLoader(config *Config, flags *pflag.FlagSet, splitter string) lib.Loader {
	if splitter == "" {
		splitter = "."
	}
	sf := func(s string) []string {
		return strings.Split(s, splitter)
	}
	return pflagP.New(config, pflagP.WithFlagSet(flags), pflagP.WithNameSplitter(sf))
}

// FlagsLoader checks the config instance to verify if the defined flags have
// been set by other providers. If not, default flag values are merged. If
// they exist, flag values are merged only if explicitly set in the command line.
func FlagsLoader(config *Config, flags *flag.FlagSet, splitter string) lib.Loader {
	if splitter == "" {
		splitter = "."
	}
	sf := func(s string) []string {
		return strings.Split(s, splitter)
	}
	return flagP.New(config, flagP.WithFlagSet(flags), flagP.WithNameSplitter(sf))
}
