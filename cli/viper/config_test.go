package viper

import (
	"bytes"
	"os"
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

var sampleConf = `
http:
  port: 8080
  idle_timeout: 10
middleware:
  cors:
    allow_credentials: true
    ignore_options: false
    allowed_headers:
      - content-type
      - x-api-key
    allowed_methods:
      - get
      - head
      - post
      - options
    allowed_origins:
      - "*"
    exposed_headers:
      - x-api-key
    max_age: 300
    options_status_code: 200
`

func TestConfigHandler(t *testing.T) {
	assert := tdd.New(t)
	opts := &ConfigOptions{
		FileName:  "config",
		FileType:  "yaml",
		Locations: []string{"testdata"},
	}

	t.Run("FromFile", func(t *testing.T) {
		os.Clearenv()

		// Empty conf handler
		conf := ConfigHandler("sample", opts)
		assert.False(conf.IsSet("http.port"))

		// Read values from file
		assert.Nil(conf.ReadFile(true), "read file")
		assert.NotEmpty(conf.FileUsed())
		assert.Equal(8080, conf.Get("http.port"))
		assert.Equal(true, conf.Get("middleware.cors.allow_credentials"))

		// Use internal viper instance for additional casting options
		vp := conf.Internals()
		assert.NotEmpty(vp.GetStringSlice("middleware.cors.allowed_headers"))

		// Override specific key with ENV variable
		assert.Nil(os.Setenv("SAMPLE_HTTP_PORT", "9191"))
		assert.Equal(9191, vp.GetInt("http.port"))

		// Unmarshal full settings
		exp := make(map[string]interface{})
		assert.Nil(conf.Unmarshal(&exp, ""), "unmarshal")
		assert.NotNil(exp["http"])
		exp = nil

		// Unmarshal settings section
		exp = make(map[string]interface{})
		assert.Nil(conf.Unmarshal(&exp, "middleware.cors"), "unmarshal")
		assert.Nil(exp["http"])
	})

	t.Run("Read", func(t *testing.T) {
		os.Clearenv()

		// Empty conf handler
		conf := ConfigHandler("sample", opts)
		assert.False(conf.IsSet("http.port"))

		// Read values from provided content
		assert.Nil(conf.Read(bytes.NewReader([]byte(sampleConf))), "read from source")
		assert.Empty(conf.FileUsed())
		assert.Equal(8080, conf.Get("http.port"))
		assert.Equal(true, conf.Get("middleware.cors.allow_credentials"))

		// Use internal viper instance for additional casting options
		vp := conf.Internals()
		assert.NotEmpty(vp.GetStringSlice("middleware.cors.allowed_headers"))

		// Override specific key with ENV variable
		assert.Nil(os.Setenv("SAMPLE_HTTP_PORT", "9191"))
		assert.Equal(9191, vp.GetInt("http.port"))

		// Unmarshal full settings
		exp := make(map[string]interface{})
		assert.Nil(conf.Unmarshal(&exp, ""), "unmarshal")
		assert.NotNil(exp["http"])
		exp = nil

		// Unmarshal settings section
		exp = make(map[string]interface{})
		assert.Nil(conf.Unmarshal(&exp, "middleware.cors"), "unmarshal")
		assert.Nil(exp["http"])
	})
}
