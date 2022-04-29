/*
Package loader provide a helper mechanism to work with settings for complex applications.

A Helper instance provide a simplified mechanism to load and apply complex
configuration settings commonly used when deploying production services. This
allows to handle configurations in YAML or JSON format to simplify storage
and sharing.

	// Create a new helper instance.
	httpMod := new(compHTTP)
	conf, _ := New(
		WithPrefix("server"),
		WithComponent("http", httpMod),
	)

	// Validate settings.
	if err := conf.Validate(); err != nil {
		panic(err)
	}

	// Register parameters on a CLI command
	_ = cli.SetupCommandParams(sampleCobraCommand, conf.Params())

	// Export settings as a YAML file for portability.
	backup, _ := yaml.Marshal(conf.Export())
	fmt.Printf("%s", backup)

	// At a later point you can to restore a helper settings from a previously
	// exported YAML file.
	restore := map[string]interface{}{}
	_ = yaml.Unmarshal(backup, &restore)
	_ = conf.Restore(restore)

Sample configuration file.

	server:
		http:
			port: 8080
			idle_timeout: 10
			tls:
				enabled: true
				system_ca: true
				cert: testdata/server.sample_cer
				key: testdata/server.sample_key
				custom_ca:
				  - testdata/ca.sample_cer
*/
package loader
