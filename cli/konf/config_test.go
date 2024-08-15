package konf

import (
	"flag"
	"os"
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

func TestKonf(t *testing.T) {
	assert := tdd.New(t)

	// ENV override
	os.Setenv("MYAPP_HTTP_PORT", "9092")

	// flag override
	flags := flag.NewFlagSet("start", flag.ContinueOnError)
	flags.Int("http.middleware.compression", 0, "gzip compression value")
	flags.Parse([]string{"--http.middleware.compression=7"})

	locations := []string{"testdata/config.yaml"}
	locations = append(locations, DefaultLocations("sample-app", "config.yaml")...)
	opts := []Option{
		WithFileLocations(locations),
		WithEnv("myapp"),
		WithFlags(flags),
	}
	config, err := Setup(opts...)
	assert.Nil(err, "load config")

	type sampleConf struct {
		Port       int `yaml:"port"`
		Timeout    int `yaml:"idle_timeout"`
		Middleware struct {
			Gzip int `yaml:"compression"`
		}
	}

	// default values
	appConf := sampleConf{
		Port:    7070,
		Timeout: 5,
	}
	assert.Nil(config.Unmarshal("http", &appConf), "unmarshal")
	assert.Equal(9092, appConf.Port, "ENV override")
	assert.Equal(10, appConf.Timeout, "file override")
	assert.Equal(7, appConf.Middleware.Gzip, "flag override")
	t.Logf("\n%s\n", config.Explain("http"))
}
