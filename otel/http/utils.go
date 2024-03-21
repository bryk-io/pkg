package otelhttp

import "net/http"

// FilterByPath will omit instrumentation for requests that match any
// of the provided paths.
func FilterByPath(paths []string) Filter {
	return func(r *http.Request) bool {
		for _, path := range paths {
			if r.URL.Path == path {
				return false
			}
		}
		return true
	}
}

// FilterByHeaders will omit instrumentation for requests that include
// any of the provided key-value pairs in their headers.
func FilterByHeaders(headers map[string]string) Filter {
	return func(r *http.Request) bool {
		for key, value := range headers {
			if r.Header.Get(key) == value {
				return false
			}
		}
		return true
	}
}
