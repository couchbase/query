//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package liner

import (
	"io"

	"github.com/couchbase/query/shell/vliner"
	pliner "github.com/peterh/liner"
)

type State struct {
	orig   *pliner.State
	vi     *vliner.State
	viMode bool
}

func NewLiner(viInput bool) *State {
	var s = State{viMode: viInput}
	if !viInput {
		s.orig = pliner.NewLiner()
	} else {
		s.vi = vliner.NewLiner()
	}
	return &s
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

func (s *State) ReadHistory(r io.Reader) (num int, err error) {
	if !s.viMode {
		return s.orig.ReadHistory(r)
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
