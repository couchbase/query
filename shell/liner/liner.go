//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package liner

import (
	"io"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/shell/vliner"
	pliner "github.com/peterh/liner"
)

type State struct {
	orig   *pliner.State
	vi     *vliner.State
	viMode bool
}

func NewLiner(viInput bool) (*State, errors.Error) {
	var s = State{viMode: viInput}
	if !viInput {
		s.orig = pliner.NewLiner()
	} else {
		var err error
		s.vi, err = vliner.NewLiner()
		if nil != err {
			return nil, errors.NewShellErrorInitTerminal(err)
		}
	}
	return &s, nil
}

func (s *State) Close() error {
	if !s.viMode {
		return s.orig.Close()
	}
	return s.vi.Close()
}

func (s *State) Prompt(p string) (string, error) {
	if !s.viMode {
		return s.orig.Prompt(p)
	}
	return s.vi.Prompt(p)
}

// This is a workaround so as not to have to fork and maintain the legacy input mechanism (peterh/liner).
// The liner state interface is closed and there isn't provision to hook custom history functions in so we're left having to
// process the I/O streams to facilitate some level of compatibility (without restricting features) between the two mechanisms.

type interceptHistoryReader struct {
	io.Reader
	src io.Reader
	buf []byte
}

// converts multi-line history entries into single line entries
// multi-line entries use ASCII 30 (record separator) to indicate newlines since the existing history format is line-based
// this means we have to translate them from ASCII 30 to something (space) that doesn't interfere with the legacy liner's rendering
// but does mean that after loading & saving with the legacy liner, all embedded newlines are lost
func (ir *interceptHistoryReader) Read(p []byte) (n int, err error) {
	i := 0
	for len(p) > i {
		_, err := ir.src.Read(ir.buf)
		if nil != err {
			return i, err
		}
		if '\x1e' == ir.buf[0] {
			ir.buf[0] = ' '
		}
		p[i] = ir.buf[0]
		i++
	}
	return i, nil
}

func (s *State) ReadHistory(r io.Reader) (num int, err error) {
	if !s.viMode {
		// expensive waste but default isn't available otherwise and peterh/liner uses the default
		// (minus 1 here else it trigers the limit)
		ir := interceptHistoryReader{src: r, buf: make([]byte, 1)}
		return s.orig.ReadHistory(&ir)
	}
	return s.vi.ReadHistory(r)
}

func (s *State) WriteHistory(w io.Writer) (num int, err error) {
	if !s.viMode {
		return s.orig.WriteHistory(w)
	}
	return s.vi.WriteHistory(w)
}

func (s *State) AppendHistory(item string) {
	if !s.viMode {
		s.orig.AppendHistory(item)
	} else {
		s.vi.AppendHistory(item)
	}
}

func (s *State) ClearHistory() {
	if !s.viMode {
		s.orig.ClearHistory()
	} else {
		s.vi.ClearHistory()
	}
}

func (s *State) SetCtrlCAborts(aborts bool) {
	if !s.viMode {
		s.orig.SetCtrlCAborts(aborts)
	} else {
		s.vi.SetCtrlCAborts(aborts)
	}
}

func (s *State) SetMultiLineMode(mlmode bool) {
	if !s.viMode {
		s.orig.SetMultiLineMode(mlmode)
	} else {
		s.vi.SetMultiLineMode(mlmode)
	}
}

func (s *State) SetCommandCallback(f func(...string) string) {
	// not implemented for pliner
	if s.viMode {
		s.vi.SetCommandCallback(f)
	}
}
