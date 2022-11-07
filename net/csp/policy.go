package csp

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Policy definitions provide a mechanism to define and enforce the security
// posture of a web application.
type Policy struct {
	reportTo   []string
	reportOnly bool
	allowEval  bool
	baseURI    string
	defaultSrc string
	nonce      string
}

// New policy with the specified options. With the generated CSP policy modern
// browsers will execute only those scripts whose nonce attribute matches the
// value set in the policy header, as well as scripts dynamically added to the
// page by scripts with the proper nonce. Older browsers, which don't support
// the CSP3 standard will fall back to regular execution without any protection
// against XSS vulnerabilities, but will allow the application to function properly.
//
// More information: https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
func New(opts ...Option) (*Policy, error) {
	p := &Policy{
		nonce:      nonce(),
		reportTo:   []string{},
		baseURI:    "'self'",
		defaultSrc: "'self'",
	}
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Refresh the policy `nonce` and return the new value. Usually this should be
// called with every page load and pass the new value to the template system
// to incorporate it on script tags.
//
//	<script nonce="{nonce}" src="/path/to/script.js"></script>
func (p *Policy) Refresh() string {
	p.nonce = nonce()
	return p.nonce
}

// Compile the policy settings and return the corresponding CSP definition.
func (p *Policy) Compile() string {
	// policy segments
	var opts []string

	// script source directives
	var ss []string

	// default-src serves as a fallback for the other CSP fetch directives
	opts = append(opts, fmt.Sprintf("default-src %s", p.defaultSrc))

	// restricts the URLs which can be used in a document's <base> element.
	opts = append(opts, fmt.Sprintf("base-uri %s", p.baseURI))

	// the nonce directive means that <script> elements will be allowed to execute
	// only if they contain a nonce attribute matching the randomly-generated value
	// which appears in the policy.
	// In the presence of a CSP nonce the unsafe-inline directive will be ignored by
	// modern browsers. Older browsers, which don't support `nonce`, will see
	// unsafe-inline and allow inline scripts to execute.
	ss = append(ss, fmt.Sprintf("'strict-dynamic' 'nonce-%s'", p.nonce))

	// allows the execution of scripts dynamically added to the page, as long as they
	// were loaded by a safe, already-trusted script.
	ss = append(ss, "'unsafe-inline' http: https:")

	// allows the application to use the eval() JavaScript function
	if p.allowEval {
		ss = append(ss, "'unsafe-eval'")
	}
	opts = append(opts, fmt.Sprintf("script-src %s", strings.Join(ss, " ")))

	// disallows using strings with DOM XSS injection sink functions, and requires
	// matching types created by Trusted Type policies.
	opts = append(opts, "require-trusted-types-for 'script'")

	// prevents fetching and executing plugin resources embedded using <object>,
	// <embed> or <applet> tags. The most common example is Flash.
	opts = append(opts, "object-src 'none'")

	// all violations to the policy to be reported to the supplied URL
	if len(p.reportTo) > 0 {
		// target details will be added to the 'report-to' header on `Handler`
		opts = append(opts, fmt.Sprintf("report-to %s", reportToTarget))
	}
	return strings.Join(opts, ";\n")
}

// Handler returns a middleware capable of enforcing the policy instance on
// every HTTP request. Usually this should be added as a middleware to an existing
// HTTP server.
func (p *Policy) Handler() func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// set the CSP definition
			w.Header().Set("Content-Security-Policy", p.Compile())

			// add CSP reports 'report-to' target
			if len(p.reportTo) > 0 {
				w.Header().Set("Report-To", sink(p.reportTo))
			}

			// don't enforce the CSP policy but still report policy violations
			if p.reportOnly {
				w.Header().Set("Content-Security-Policy-Report-Only", "true")
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

const (
	reportToTarget = "csp-issues"
)

type reportSink struct {
	Name      string              `json:"name"`
	MaxAge    uint                `json:"max_age"`
	Endpoints []map[string]string `json:"endpoints"`
}

func nonce() string {
	nonce := make([]byte, 8)
	_, _ = rand.Read(nonce)
	return fmt.Sprintf("%x", nonce)
}

func sink(endpoints []string) string {
	rs := reportSink{
		Name:      reportToTarget,
		MaxAge:    10886400,
		Endpoints: []map[string]string{},
	}
	for _, ep := range endpoints {
		rs.Endpoints = append(rs.Endpoints, map[string]string{"url": ep})
	}
	js, _ := json.Marshal(rs)
	return string(js)
}
