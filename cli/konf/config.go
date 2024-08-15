package konf

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"strings"

	lib "github.com/nil-go/konf"
	"github.com/nil-go/konf/provider/env"
	"github.com/nil-go/konf/provider/file"
	flagP "github.com/nil-go/konf/provider/flag"
	pflagP "github.com/nil-go/konf/provider/pflag"
	"gopkg.in/yaml.v3"
)

// Config reads configuration from appropriate sources.
//
// To create a new Config, call [Setup].
type Config = lib.Config

// Setup returns a new configuration handler instance.
func Setup(opts ...Option) (config *Config, err error) {
	ss := new(settings)
	for _, opt := range opts {
		if err = opt(ss); err != nil {
			return nil, err
		}
	}

	// * setup providers; override order is important
	// * defaults -> config file -> ENV -> flags

	// setup file provider
	if len(ss.locations) > 0 {
		config, err = loadFile(ss.locations, ss.tagName)
		if err != nil {
			return nil, err
		}
	}

	// setup ENV provider
	if ss.env {
		ns := customSplitterFunc(ss.envPrefix)
		err := config.Load(env.New(env.WithPrefix(ss.envPrefix), env.WithNameSplitter(ns)))
		if err != nil {
			return nil, err
		}
	}

	// setup flags provider
	if ss.flags != nil {
		loader := flagP.New(config, flagP.WithFlagSet(ss.flags))
		if err := config.Load(loader); err != nil {
			return nil, err
		}
	}

	// setup pflags provider
	if ss.pflags != nil {
		loader := pflagP.New(config, pflagP.WithFlagSet(ss.pflags))
		if err := config.Load(loader); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// Attempt to load a configuration file from one of the provided locations. The
// function will stop on the first one that succeeds. The proper unmarshal method
// will be selected based on the file extension. The tag name used to load values
// into structs will be a default name based on the file extension or the one
// explicitly provided, if any.
func loadFile(locations []string, tag string) (*lib.Config, error) {
	var config *lib.Config
	found := false
	for _, cf := range locations {
		info, err := os.Stat(cf)
		if err != nil || info.IsDir() {
			continue // try next file
		}
		tagName, mf, err := getUnmarshal(path.Ext(info.Name()))
		if err != nil {
			continue // try next file
		}
		if tag != "" {
			tagName = tag // use the explicitly provided value
		}
		config = lib.New(lib.WithTagName(tagName))
		err = config.Load(file.New(cf, file.WithUnmarshal(mf)))
		if err == nil {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("no valid config file found")
	}
	return config, nil
}

// Function to process ENV variables names.
func customSplitterFunc(prefix string) func(string) []string {
	return func(s string) []string {
		if prefix != "" {
			s = strings.TrimPrefix(s, prefix)
		}
		return strings.Split(s, "_")
	}
}

// Return the proper unmarshal function based on the provided file extension.
func getUnmarshal(extension string) (tag string, mf func([]byte, any) error, err error) {
	switch extension {
	case ".yaml":
		return "yaml", yaml.Unmarshal, nil
	case ".yml":
		return "yaml", yaml.Unmarshal, nil
	case ".json":
		return "json", json.Unmarshal, nil
	}
	return "", nil, errors.New("unsupported file format")
}
