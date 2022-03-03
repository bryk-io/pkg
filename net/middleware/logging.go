package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	xlog "go.bryk.io/pkg/log"
)

// Logging produce output for the processed HTTP requests tagged with
// standard ECS details by default. Fields can be extended by providing
// a hook function.
func Logging(ll xlog.Logger, hook LoggingHook) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Process request
			start := time.Now().UTC()
			lrw := &loggingRW{
				ResponseWriter: w,
				code:           http.StatusOK,
				size:           0,
			}
			next.ServeHTTP(lrw, r)
			end := time.Now().UTC()
			lapse := end.Sub(start)

			// Get message details
			fields := getFields(r)
			fields.Set("duration", lapse.String())
			fields.Set("duration_ms", fmt.Sprintf("%.3f", lapse.Seconds()*1000))
			fields.Set("event.start", start.Nanosecond())
			fields.Set("event.end", end.Nanosecond())
			fields.Set("event.duration", lapse.Nanoseconds())
			fields.Set("http.response.status_code", lrw.code)
			fields.Set("http.response.body.bytes", lrw.size)
			if hook != nil {
				hook(&fields, *r)
			}

			// Log message
			ll.WithFields(fields).Print(getLevel(lrw.code), r.URL.String())
		}
		return http.HandlerFunc(fn)
	}
}

// LoggingHook provides a mechanism to extend a message fields just
// before is submitted.
type LoggingHook func(fields *xlog.Fields, r http.Request)

// Custom response writer to collect additional details.
type loggingRW struct {
	http.ResponseWriter
	size int
	code int
}

func (lrw *loggingRW) WriteHeader(code int) {
	lrw.code = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingRW) Write(content []byte) (int, error) {
	s, err := lrw.ResponseWriter.Write(content)
	if err == nil {
		lrw.size += s
	}
	return s, err
}

func getLevel(status int) xlog.Level {
	switch {
	// Server errors
	case status >= 500:
		return xlog.Error
	// User errors
	case status >= 400 && status <= 499:
		return xlog.Warning
	// Redirection
	case status >= 300 && status <= 399:
		return xlog.Debug
	// Success
	case status >= 200 && status <= 299:
		return xlog.Info
	// Informational
	case status >= 100 && status <= 199:
		return xlog.Debug
	// Unknown codes
	default:
		return xlog.Warning
	}
}

func getFields(r *http.Request) xlog.Fields {
	data := xlog.Fields{
		"user_agent.original":     r.UserAgent(),
		"client.ip":               getIP(r),
		"client.packets":          r.ContentLength,
		"http.version":            r.Proto,
		"http.request.method":     strings.ToLower(r.Method),
		"http.request.body.bytes": r.ContentLength,
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		data.Set("http.request.referrer", ref)
	}
	return data
}

func getIP(r *http.Request) (ip string) {
	if forwarded := r.Header.Get("X-Forwarded-For"); len(forwarded) > 0 {
		ip = forwarded
		return
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return
}
