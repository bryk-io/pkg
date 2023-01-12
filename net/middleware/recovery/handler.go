package recovery

import (
	"fmt"
	"net/http"
)

// Handler allows the server to convert unhandled panic events into an
// `internal server error`. This will prevent the server from crashing if a
// handler produces a `panic` operation.
func Handler() func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(fmt.Sprintf("%s", v)))
				}
			}()
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
