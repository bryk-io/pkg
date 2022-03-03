package middleware

import (
	"net/http"

	gmw "github.com/gorilla/handlers"
)

// CORS provides a "Cross Origin Resource Sharing" middleware.
func CORS(options CORSOptions) func(http.Handler) http.Handler {
	return gmw.CORS(options.parse()...)
}

// CORSOptions allow to adjust the behavior on the CORS middleware.
type CORSOptions struct {
	// Specify the user agent may pass authentication details along
	// with the request.
	AllowCredentials bool `json:"allow_credentials" yaml:"allow_credentials" mapstructure:"allow_credentials"`

	// Causes the CORS middleware to ignore OPTIONS requests, instead
	// passing them through to the next handler. This is useful when
	// your application or framework has a pre-existing mechanism for
	// responding to OPTIONS requests.
	IgnoreOptions bool `json:"ignore_options" yaml:"ignore_options" mapstructure:"ignore_options"`

	// Adds the provided headers to the list of allowed headers in a CORS
	// request. This is an append operation so the headers Accept,
	// Accept-Language, and Content-Language are always allowed. Content-Type
	// must be explicitly declared if accepting Content-Types other than
	// application/x-www-form-urlencoded, multipart/form-data, or text/plain.
	AllowedHeaders []string `json:"allowed_headers" yaml:"allowed_headers" mapstructure:"allowed_headers"`

	// Explicitly allow methods in the Access-Control-Allow-Methods header. This
	// is a replacement operation so you must also pass GET, HEAD, and POST if
	// you wish to support those methods.
	AllowedMethods []string `json:"allowed_methods" yaml:"allowed_methods" mapstructure:"allowed_methods"`

	// Sets the allowed origins for CORS requests, as used in the
	// 'Allow-Access-Control-Origin' HTTP header. Note: Passing in a
	// []string{"*"} will allow any domain
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowed_origins" mapstructure:"allowed_origins"`

	// Specify headers that are available and will not be stripped out by the
	// user-agent.
	ExposedHeaders []string `json:"exposed_headers" yaml:"exposed_headers" mapstructure:"exposed_headers"`

	// Determines the maximum age (in seconds) between preflight requests. A
	// maximum of 10 minutes is allowed. An age above this value will default
	// to 10 minutes.
	MaxAge uint `json:"max_age" yaml:"max_age" mapstructure:"max_age"`

	// Sets a custom status code on the OPTIONS requests. Default behavior
	// sets it to 200 to reflect best practices.
	OptionsStatusCode int `json:"options_status_code" yaml:"options_status_code" mapstructure:"options_status_code"`

	// Sets a function for evaluating allowed origins in CORS requests, represented
	// by the 'Allow-Access-Control-Origin' HTTP header.
	OriginValidator func(string) bool `json:"-" yaml:"-"`
}

func (co *CORSOptions) parse() []gmw.CORSOption {
	var list []gmw.CORSOption
	if co.AllowCredentials {
		list = append(list, gmw.AllowCredentials())
	}
	if co.IgnoreOptions {
		list = append(list, gmw.IgnoreOptions())
	}
	if len(co.AllowedHeaders) != 0 {
		list = append(list, gmw.AllowedHeaders(co.AllowedHeaders))
	}
	if len(co.AllowedMethods) != 0 {
		list = append(list, gmw.AllowedMethods(co.AllowedMethods))
	}
	if len(co.AllowedOrigins) != 0 {
		list = append(list, gmw.AllowedOrigins(co.AllowedOrigins))
	}
	if len(co.ExposedHeaders) != 0 {
		list = append(list, gmw.ExposedHeaders(co.ExposedHeaders))
	}
	if co.MaxAge != 0 {
		list = append(list, gmw.MaxAge(int(co.MaxAge)))
	}
	if co.OptionsStatusCode != 0 {
		list = append(list, gmw.OptionStatusCode(co.OptionsStatusCode))
	}
	if co.OriginValidator != nil {
		list = append(list, gmw.AllowedOriginValidator(co.OriginValidator))
	}
	return list
}
