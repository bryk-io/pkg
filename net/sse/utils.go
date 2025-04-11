package sse

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// SplitFunc to be used by a buffer scanner looking for SSE events on a
// data stream; usually an HTTP response body.
func scanForEvents(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// no more data to process
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	// We have a full event payload to parse.
	if i, nlen := containsSeparator(data); i >= 0 {
		return i + nlen, data[0:i], nil
	}
	// If we're at EOF, we have all the data.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// Returns a tuple containing the index of a double newline, and the number
// of bytes represented by that sequence. If no double newline is present,
// the first value will be negative.
func containsSeparator(data []byte) (int, int) {
	// Search for each potentially valid sequence of newline characters
	crcr := bytes.Index(data, []byte("\r\r"))
	lflf := bytes.Index(data, []byte("\n\n"))
	crlflf := bytes.Index(data, []byte("\r\n\n"))
	lfcrlf := bytes.Index(data, []byte("\n\r\n"))
	crlfcrlf := bytes.Index(data, []byte("\r\n\r\n"))
	// Find the earliest position of a double newline combination
	minPos := minPosInt(crcr, minPosInt(lflf, minPosInt(crlflf, minPosInt(lfcrlf, crlfcrlf))))
	// Determine the length of the sequence
	var nlen int
	switch minPos {
	case crlfcrlf:
		nlen = 4
	case crlflf, lfcrlf:
		nlen = 3
	default:
		nlen = 2
	}
	return minPos, nlen
}

// Returns the minimum non-negative value out of the two values. If both
// are negative, a negative value is returned.
func minPosInt(a, b int) int {
	if a < 0 {
		return b
	}
	if b < 0 {
		return a
	}
	if a > b {
		return b
	}
	return a
}

// Decode an event instance from the provided byte array.
func parseEvent(data []byte) Event {
	lf := []byte("\n")[0]
	if data[len(data)-1] != lf {
		data = append(data, lf) // attach final line break if required
	}
	var lines []string
	rd := bufio.NewReader(bytes.NewReader(data))
	for {
		l, err := rd.ReadString(lf)
		if err == nil {
			lines = append(lines, l)
			continue
		}
		break
	}
	ev := Event{}
	for _, l := range lines {
		k := strings.SplitN(l, ":", 2)
		if len(k) != 2 {
			continue // invalid line
		}
		val := strings.TrimSpace(k[1])
		switch k[0] {
		case "event":
			ev.name = val
		case "id":
			id, err := strconv.Atoi(val)
			if err == nil {
				ev.id = id
			}
		case "retry":
			retry, err := strconv.Atoi(val)
			if err == nil {
				ev.retry = uint(retry)
			}
		case "data":
			_ = json.Unmarshal([]byte(val), &ev.data)
		}
	}
	return ev
}
