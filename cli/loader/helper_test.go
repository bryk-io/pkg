package loader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/cli"
	httpLoader "go.bryk.io/pkg/cli/loader/http"
	otelLoader "go.bryk.io/pkg/cli/loader/otel"
	rpcLoader "go.bryk.io/pkg/cli/loader/rpc"
	"go.bryk.io/pkg/net/http"
	"gopkg.in/yaml.v3"
)

var sampleCobraCommand = &cobra.Command{}

func TestHelper(t *testing.T) {
	assert := tdd.New(t)

	// Create new helper instance
	h, err := New(WithPrefix("server"))
	assert.Nil(err, "create new helper instance")
	assert.Nil(h.Validate(), "empty validate")
	assert.Empty(h.Params(), "empty params")

	// Add sample component
	seg := httpLoader.New()
	seg.TLS.Enabled = true
	seg.TLS.SystemCA = true
	seg.TLS.Cert = "testdata/server.sample_cer"
	seg.TLS.Key = "testdata/server.sample_key"
	seg.TLS.CustomCA = []string{"testdata/ca.sample_cer"}
	h.Register("http", seg)
	h.Register("otel", otelLoader.New())
	h.Register("rpc", rpcLoader.New())
	assert.Nil(h.Validate(), "validate")
	assert.NotEmpty(h.Params(), "params")

	// Register parameters on a sample command
	err = cli.SetupCommandParams(sampleCobraCommand, h.Params(), nil)
	assert.Nil(err, "setup command")
	t.Logf("\n%+v", sampleCobraCommand.Flags().FlagUsages())

	// Get exportable data structure
	data := h.Export()

	// YAML encode helper settings
	yl, err := yaml.Marshal(data)
	assert.Nil(err, "yaml encode")
	t.Logf("\n%s", yl)

	// JSON encode helper settings
	js, err := json.MarshalIndent(data, "", "  ")
	assert.Nil(err, "json encode")
	t.Logf("\n%s", js)

	// Load helper settings with Viper (common use case)
	conf := viper.New()
	conf.SetConfigType("yaml")
	assert.Nil(conf.ReadConfig(bytes.NewReader(yl)), "read config")
	assert.Equal(8080, conf.GetInt("server.http.port"), "port")
	assert.Equal(5, conf.GetInt("server.http.idle_timeout"), "idle_timeout")
	assert.Equal(seg.TLS.Enabled, conf.GetBool("server.http.tls.enabled"))
	assert.Equal(seg.TLS.CustomCA, conf.GetStringSlice("server.http.tls.custom_ca"))

	// Expand settings for a specific component
	httpParams := h.Expand("http")
	assert.NotNil(httpParams, "expand http")
	list, ok := httpParams.([]http.Option)
	assert.True(ok, "wrong params type")
	assert.Equal(3, len(list), "invalid params count")

	// Load settings from a viper instance (common use case)
	h2, err := New(
		WithPrefix("server"),
		WithComponent("http", seg),
		WithComponent("otel", otelLoader.New()),
		WithComponent("rpc", rpcLoader.New()),
	)
	assert.Nil(err, "create second helper")
	assert.Nil(h2.Restore(conf.AllSettings()), "load restored settings")
	yl2, err := yaml.Marshal(h2.Export())
	assert.Nil(err, "get data from 2nd helper instance")
	assert.Equal(yl, yl2, "invalid restored contents")

	t.Run("NoPrefix", func(t *testing.T) {
		// Create new helper instance
		h3, err := New()
		assert.Nil(err, "create new helper instance")
		assert.Nil(h3.Validate(), "empty validate")
		assert.Empty(h3.Params(), "empty params")
		h3.Register("http", seg)
		h3.Register("otel", otelLoader.New())

		// YAML encode helper settings
		yl, err := yaml.Marshal(h3.Export())
		assert.Nil(err, "yaml encode")
		t.Logf("\n%s", yl)

		// Load helper settings with Viper (common use case)
		conf := viper.New()
		conf.SetConfigType("yaml")
		assert.Nil(conf.ReadConfig(bytes.NewReader(yl)), "read config")
		assert.Equal(8080, conf.GetInt("http.port"), "port")
		assert.Equal(5, conf.GetInt("http.idle_timeout"), "idle_timeout")
		assert.Equal(seg.TLS.Enabled, conf.GetBool("http.tls.enabled"))
		assert.Equal(seg.TLS.CustomCA, conf.GetStringSlice("http.tls.custom_ca"))
	})
}

func ExampleNew() {
	// Error checks omit for brevity.

	// Create a new helper instance.
	conf, _ := New(
		WithPrefix("server"),
		WithComponent("http", httpLoader.New()),
		WithComponent("otel", otelLoader.New()),
	)

	// Validate settings.
	if err := conf.Validate(); err != nil {
		panic(err)
	}

	// Register parameters on a CLI command
	_ = cli.SetupCommandParams(sampleCobraCommand, conf.Params(), nil)

	// Export settings as a YAML file for portability.
	backup, _ := yaml.Marshal(conf.Export())
	fmt.Printf("%s", backup)

	// At a later point you can to restore a helper settings from a previously
	// exported YAML file.
	restore := map[string]interface{}{}
	_ = yaml.Unmarshal(backup, &restore)
	_ = conf.Restore(restore)
}
