//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package pager

import (
	"bufio"
	"io"
	"os"
	"runtime"

	"golang.org/x/term"
)

type output struct {
	w   io.Writer
	cmd bool
}

type Pager struct {
	w       []*output
	height  int
	width   int
	line    int
	col     int
	paging  bool
	skip    bool
	newline string
}

func NewPager() *Pager {
	this := &Pager{}

	if runtime.GOOS == "windows" {
		this.newline = "\r\n"
	} else {
		this.newline = "\n"
	}

	return this
}

func (this *Pager) SetPaging(on bool) {
	this.paging = on
	if this.paging {
		this.setSize()
	}
}

func (this *Pager) setSize() {
	var err error
	this.width, this.height, err = term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		this.height = 23
		this.width = 80
	} else {
		this.height--
	}
}

func (this *Pager) Reset() {
	this.line = 0
	this.col = 0
	this.setSize()
	this.skip = false
}

var msg = "-- (c)ontinue, (s)stop paging, (q)uit --"
var clr = "\r                                        \r"

func (this *Pager) Continue() bool {
	os.Stdout.WriteString(msg)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		oldState = nil
	}
	rv := false
	inp := bufio.NewReader(os.Stdin)
cont:
	for {
		r, sz, err := inp.ReadRune()
		if err != nil && err != io.EOF {
			return false
		}
		if sz > 0 {
			switch r {
			case ' ':
				rv = true
				break cont
			case 'C':
				fallthrough
			case 'c':
				rv = true
				break cont
			case 'Q':
				fallthrough
			case 'q':
				rv = false
				break cont
			case '\r':
				rv = true
				this.line = this.height - 1
				break cont
			case 'S':
				fallthrough
			case 's':
				this.skip = true
				rv = true
				break cont
			}
		}
	}
	if oldState != nil {
		term.Restore(int(os.Stdin.Fd()), oldState)
	}
	os.Stdout.WriteString(clr)
	return rv
}

// only to targets listed as wanting commands
func (this *Pager) WriteCommand(cmd string) (int, error) {
	cmd = cmd + this.newline
	for i := range this.w {
		if this.w[i].cmd {
			_, err := this.w[i].w.Write([]byte(cmd))
			if err != nil {
				return 0, err
			}
		}
	}
	return len(cmd), nil
}

// only to targers NOT listed as wanting commands
func (this *Pager) EchoCommand(cmd string) (int, error) {
	cmd = cmd + this.newline
	for i := range this.w {
		if !this.w[i].cmd {
			_, err := this.w[i].w.Write([]byte(cmd))
			if err != nil {
				return 0, err
			}
		}
	}
	return len(cmd), nil
}

func (this *Pager) WriteString(s string) (int, error) {
	return this.Write([]byte(s))
}

func (this *Pager) copyToOutput(buf []byte) (int, error) {
	for i := range this.w {
		_, err := this.w[i].w.Write(buf)
		if err != nil {
			return 0, err
		}
	}
	return len(buf), nil
}

func (this *Pager) Write(buf []byte) (int, error) {
	if len(this.w) == 0 {
		return 0, io.EOF
	}
	if !this.paging || this.skip || len(this.w) == 1 && this.w[0].w != os.Stdout {
		return this.copyToOutput(buf)
	}
	for n := range buf {
		if this.skip {
			sn, err := this.copyToOutput(buf[n:])
			if err == nil {
				n += sn
			}
			return n, err
		}
		for i := range this.w {
			_, err := this.w[i].w.Write(buf[n : n+1])
			if err != nil {
				return 0, err
			}
		}
		if buf[n] == '\n' {
			this.col = 0
			this.line++
			if this.line >= int(this.height) {
				this.line = 0
				this.col = 0
				if !this.Continue() {
					return 0, io.EOF
				}
			}
		} else {
			this.col++
			if this.col == int(this.width) {
				this.col = 0
				this.line++
				if this.line >= int(this.height) {
					this.line = 0
					if !this.Continue() {
						return 0, io.EOF
					}
				}
			}
		}
	}
	return len(buf), nil
}

func (this *Pager) AddOutput(w io.Writer, cmd bool) {
	for i := range this.w {
		if this.w[i].w == w {
			this.w[i].cmd = cmd
			return
		}
	}
	this.w = append(this.w, &output{w, cmd})
}

func (this *Pager) SetOutput(w io.Writer, cmd bool) {
	this.w = append([]*output(nil), &output{w, cmd})
}

func (this *Pager) Flush() {
	for i := range this.w {
		if s, ok := this.w[i].w.(interface{ Sync() error }); ok {
			s.Sync()
		}
	}
}
