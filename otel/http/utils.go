package otelhttp

import "slices"

import "net/http"

// FilterByPath will omit instrumentation for requests that match any
// of the provided paths.
func FilterByPath(paths []string) Filter {
	return func(r *http.Request) bool {
		return !slices.Contains(paths, r.URL.Path)
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
