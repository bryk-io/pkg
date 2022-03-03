/*
Package loader provide a helper mechanism to work with complex settings for network components.

A Helper instance provide a simplified mechanism to load and apply complex
configuration settings commonly used when deploying production network services.
This allows to handle configurations in YAML or JSON format to simplify storage
and sharing, for example using CVS.

To use it on CLI-based applications.

	// Enable segments to display CLI params
	helper := New()
	segments := []string{
		SegmentRPC,
		SegmentGraphQL,
		SegmentWebsocket,
		SegmentObservability,
		SegmentMiddlewareHSTS,
		SegmentMiddlewareCORS,
		SegmentMiddlewareMetadata,
	}

	// Use the helper instance to setup CLI command
	_ = cli.SetupCommandParams(sampleCobraCommand, helper.Params(segments...))

	// At a later point values can accessed using viper for example
	_ = viper.Unmarshal(helper.Data)

Sample configuration file.

	graphql:
	  max_complexity: 3000
	  introspection: true
	  options_enabled: true
	  get_enabled: true
	  post_enabled: true
	  apollo_tracing: false
	  query_cache: 512
	  apq: 100
	  subscription_origins:
	    - "*"
	  multipart:
	    enabled: false
	    max_upload_size: 10
	    max_memory: 5
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
	  hsts:
	    max_age: 8760
	    host_override: ""
	    accept_forwarded_proto: true
	    send_preload_directive: false
	    include_subdomains: false
	  metadata:
	    headers:
	      - x-api-key
	observability:
	  filtered_methods:
	    - sample.ServiceAPI/Ping
	  tracer_name: "custom_tracer/name"
	  service_name: "sample_service"
	  service_version: "0.1.0"
	  attributes: {}
	  global: true
	  log_json: true
	rpc:
	  input_validation: true
	  network_interface: all
	  port: 9999
	  unix_socket: ""
	  http_gateway:
	    enabled: false
	    port: 9999
	  resource_limits:
	    connections: 1000
	    requests: 50
	    rate: 5000
	  tls:
	    enabled: true
	    system_ca: true
	    cert: testdata/server.sample_cer
	    key: testdata/server.sample_key
	    custom_ca:
	      - testdata/ca.sample_cer
	    auth_by_certificate:
	      - testdata/ca.sample_cer
	websocket:
	  compression: true
	  handshake_timeout: 5
	  method_override: ""
	  authorization_cookie: "session_credentials"
	  sub_protocols:
	    - graphql-ws
	  forward_headers:
	    - x-api-key

*/
package loader
