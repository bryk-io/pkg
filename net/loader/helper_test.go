package loader

import (
	"io/ioutil"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/rpc"
)

var sampleCobraCommand = &cobra.Command{}

func TestHelper(t *testing.T) {
	assert := tdd.New(t)

	// Enable segments to display CLI params
	helper := New()
	segments := []string{
		SegmentRPC,
		SegmentHTTP,
		SegmentWebsocket,
		SegmentObservability,
		SegmentMiddlewareHSTS,
		SegmentMiddlewareCORS,
		SegmentMiddlewareMetadata,
	}

	// Use the helper instance to set up CLI command
	err := cli.SetupCommandParams(sampleCobraCommand, helper.Params(segments...))
	assert.Nil(err, "setup command")

	// At a later point values can be accessed using viper for example
	err = viper.Unmarshal(helper.Data)
	assert.Nil(err, "viper unmarshal")

	// Inspect produced usage information
	// t.Log("\n" + sampleCobraCommand.Flags().FlagUsages())
}

func ExampleFromYAML() {
	// Load configuration file
	yl, _ := ioutil.ReadFile("testdata/conf.yaml")
	h, err := FromYAML(yl)
	if err != nil {
		panic(err)
	}

	// Start RPC server
	server, _ := rpc.NewServer(h.ServerRPC()...)
	_ = server.Start(nil)
}

func ExampleNew() {
	// Create new helper instance and enable segments to display CLI params
	helper := New()
	segments := []string{
		SegmentRPC,
		SegmentHTTP,
		SegmentWebsocket,
		SegmentObservability,
		SegmentMiddlewareHSTS,
		SegmentMiddlewareCORS,
		SegmentMiddlewareMetadata,
	}

	// Use the helper instance to setup CLI command
	err := cli.SetupCommandParams(sampleCobraCommand, helper.Params(segments...))
	if err != nil {
		panic(err)
	}

	// At a later point values can accessed using viper for example
	err = viper.Unmarshal(helper.Data)
	if err != nil {
		panic(err)
	}
}
