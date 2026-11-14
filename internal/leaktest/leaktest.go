// Package leaktest provides goroutine leak checks for tests using the
// runtime's goroutine leak profile.
//
// The goroutineleak profile was introduced in Go 1.27. When running under an
// older toolchain the profile is unavailable and leak checks are skipped.
package leaktest

import (
	"bytes"
	"fmt"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	profileName  = "goroutineleak"
	retryPeriod  = 10 * time.Millisecond
	retryTimeout = 100 * time.Millisecond
)

// VerifyTestMain runs the test suite and exits with a non-zero status when
// leaked goroutines are reported after all tests have completed. It is meant
// to be called from a package's `TestMain` function.
func VerifyTestMain(m *testing.M) {
	code := m.Run()
	count, report, ok := check()
	if ok && report != "" {
		fmt.Fprintf(os.Stderr, "leaktest: found %d leaked goroutine(s)\n%s", count, report)
		code = 1
	}
	os.Exit(code)
}

// VerifyNone fails the test when leaked goroutines are reported. It is meant
// to be used at the end of a single test, usually with defer.
func VerifyNone(t testing.TB) {
	t.Helper()
	count, report, ok := check()
	if !ok || report == "" {
		return
	}
	t.Fatalf("leaktest: found %d leaked goroutine(s)\n%s", count, report)
}

// check writes the goroutine leak profile and retries briefly to avoid
// reporting goroutines that are in the process of terminating. The boolean
// return value reports whether the profile is available.
func check() (int, string, bool) {
	profile := pprof.Lookup(profileName)
	if profile == nil {
		return 0, "", false
	}

	deadline := time.Now().Add(retryTimeout)
	for {
		var buf bytes.Buffer
		if err := profile.WriteTo(&buf, 1); err != nil {
			return 0, fmt.Sprintf("leaktest: error writing %s profile: %v", profileName, err), true
		}
		text := buf.String()
		if count := leakedCount(text); count == 0 {
			return 0, "", true
		} else if !time.Now().Before(deadline) {
			return count, text, true
		}
		time.Sleep(retryPeriod)
	}
}

// leakedCount extracts the number of leaked goroutines from the text
// representation of a goroutine leak profile.
func leakedCount(text string) int {
	prefix := profileName + " profile: total "
	for line := range strings.Lines(text) {
		value, found := strings.CutPrefix(line, prefix)
		if !found {
			continue
		}
		count, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return count
		}
	}
	return 0
}
