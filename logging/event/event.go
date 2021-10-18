//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package event

import (
	"os"
	"strings"
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

// event IDs must be in the range 1024-2047 (incl.)
type EventType uint16

const (
	CRASH          EventType = 1024
	CONFIG_CHANGE  EventType = 1025
	QUOTA_EXCEEDED EventType = 1026
)

var eventDescription = map[EventType]string{
	CRASH:          "Service crashed",
	CONFIG_CHANGE:  "Configuration changed",
	QUOTA_EXCEEDED: "Request memory quota exceeded",
}

type EventLevel string

const (
	INFO    EventLevel = "info"
	ERROR   EventLevel = "error"
	WARNING EventLevel = "warn"
	FATAL   EventLevel = "fatal"
)

const _EVENT_TIMESTAMP_FORMAT = "2006-01-02T15:04:05.000Z"

const _COMPONENT = "query"

const MaxRecLen = 3072

var configStore clustering.ConfigurationStore

func Init(c clustering.ConfigurationStore) {
	configStore = c
}

func writeValue(buf *[]byte, title string, value interface{}) bool {
	mv, _ := json.Marshal(value)
	if len(mv)+len(title)+4 > MaxRecLen-2-len(*buf) {
		return false
	}
	if len(*buf) > 0 && (*buf)[len(*buf)-1] != '{' {
		*buf = append(*buf, byte(','))
	}
	*buf = append(*buf, byte('"'))
	*buf = append(*buf, []byte(title)...)
	*buf = append(*buf, []byte("\":")...)
	*buf = append(*buf, mv...)
	return true
}

func Report(e EventType, l EventLevel, args ...interface{}) {

	body := make([]byte, 0, 3072)

	uuid, _ := util.UUIDV4()
	body = append(body, byte('{'))
	writeValue(&body, "uuid", uuid)
	writeValue(&body, "component", _COMPONENT)
	writeValue(&body, "event_id", e)
	writeValue(&body, "description", eventDescription[e])
	writeValue(&body, "severity", l)
	writeValue(&body, "timestamp", time.Now().UTC().Format(_EVENT_TIMESTAMP_FORMAT))

	if len(args) > 0 {
		body = append(body, []byte(",\"extra_attributes\":{")...)
		// expect pairs else panic
		if len(args)%2 == 1 {
			panic("Invalid arguments to logging.Event")
		}
		for i := 0; i < len(args); i += 2 {
			// each pair must have a string key as the first element
			k, ok := args[i].(string)
			if !ok {
				panic("Invalid arguments to logging.Event")
			}
			if !writeValue(&body, k, args[i+1]) {
				logging.Errorf("Event record exceeded maximum length. Dropping extra_attribute: %v (%v).", k, args[i+1])
			}
		}
		body = append(body, byte('}'))
	}
	body = append(body, byte('}'))

	if configStore == nil {
		logging.Errorf("Events not initialised. Failed to report event: %v", string(body))
		return
	}
	if s, _ := configStore.State(); s == clustering.STANDALONE {
		return
	}
	cl, err := configStore.Cluster()
	if cl == nil || err != nil {
		logging.Errorf("Cluster not found (%v). Failed to report event: %v", err, string(body))
		return
	}

	cl.ReportEventAsync(string(body))
}

// reduce the number of characters needed to represent a stack by stripping arguments and paths and including only lines referring
// to functions (strip goroutine number and created by lines)
func CompactStack(raw string, max int) []string {
	lines := strings.Split(raw, "\n")
	output := make([]rune, 0, len(raw))
	stack := make([]string, 0, 10)
	for i := 0; i < len(lines) && max > 0; i++ {
		if len(lines[i]) == 0 {
			continue
		}
		l := lines[i]
		if l[0] != '\t' {
			n := -1
			if l[len(l)-1] == ')' {
				n = strings.LastIndexByte(l, '(')
			}
			if n != -1 {
				l = l[:n]
				n = strings.LastIndexByte(l, os.PathSeparator)
				if n != -1 {
					l = l[n+1:]
				}
				output = append(output, []rune(strings.TrimSpace(l))...)
			}
		} else if len(output) > 0 {
			n := strings.LastIndexByte(l, os.PathSeparator)
			if n != -1 {
				l = l[n+1:]
			}
			n = strings.LastIndexByte(l, '+')
			if n != -1 {
				output = append(output, []rune(l[n:])...)
				l = l[:n]
			}
			output = append(output, '[')
			output = append(output, []rune(strings.TrimSpace(l))...)
			output = append(output, ']')
			if len(output) > max {
				output = output[:max]
			}
			if len(output) > 0 {
				stack = append(stack, string(output))
				max -= len(output)
				output = output[:0]
			}
		}
	}
	return stack
}

func UpTo(s string, l int) string {
	if l > 3 && len(s) > l {
		s = s[:l-3] + "..."
	}
	// look for unclosed redaction tag; add close which may push length over desired length
	n := strings.LastIndex(s, "<ud>")
	if n != -1 {
		if strings.LastIndex(s[n:], "</ud>") == -1 {
			s = s + "</ud>"
		}
	}
	return s
}
