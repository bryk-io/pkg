package log

import (
	"strings"

	"go.bryk.io/pkg/metadata"
)

func lPrint(ll SimpleLogger, lv Level, args ...any) {
	switch lv {
	case Debug:
		ll.Debug(args...)
	case Info:
		ll.Info(args...)
	case Warning:
		ll.Warning(args...)
	case Error:
		ll.Error(args...)
	case Panic:
		ll.Panic(args...)
	case Fatal:
		ll.Fatal(args...)
	}
}

func lPrintf(ll SimpleLogger, lv Level, format string, args ...any) {
	switch lv {
	case Debug:
		ll.Debugf(format, args...)
	case Info:
		ll.Infof(format, args...)
	case Warning:
		ll.Warningf(format, args...)
	case Error:
		ll.Errorf(format, args...)
	case Panic:
		ll.Panicf(format, args...)
	case Fatal:
		ll.Fatalf(format, args...)
	}
}

func sanitize(args ...any) []any {
	var (
		vs string
		ok bool
		sv = make([]any, len(args))
	)
	for i, v := range args {
		// remove all newlines and carriage returns
		if vs, ok = v.(string); ok {
			v = strings.ReplaceAll(strings.ReplaceAll(vs, "\n", ""), "\r", "")
		}
		sv[i] = v
	}
	return sv
}

func fields(md ...metadata.MD) []any {
	// get all fields from the metadata
	fields := metadata.New()
	fields.Join(md...)
	values := fields.Values()

	// ensure max number of fields is not exceeded
	size := len(values) * 2
	if size > maxFields {
		size = maxFields
	}

	// build the list of fields
	i := 0
	list := make([]any, size)
	for k, v := range values {
		list[i] = k
		list[i+1] = v
		i += 2
	}
	return list
}
