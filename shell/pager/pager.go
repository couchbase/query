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
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type output struct {
	w   io.Writer
	cmd bool
}

type Pager struct {
	w         []*output
	height    int
	width     int
	line      int
	col       int
	paging    bool
	skip      bool
	newline   string
	skipToEnd bool
	lastPage  [][]byte
	lpStart   int
	lpNext    int
	lpCont    bool
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

func (this *Pager) Reset(skipToEnd bool) {
	this.line = 0
	this.col = 0
	this.setSize()
	this.skip = false
	this.skipToEnd = skipToEnd
	this.lastPage = nil
	this.lpStart = 0
	this.lpNext = 0
}

var msg = "-- (c)ontinue, (s)stop paging, (q)uit --"
var msgWithSkip = "-- (c)ontinue, (s)stop paging, s(k)ip to end, (q)uit --"
var clr = "\r                                                       \r"

func (this *Pager) Continue() bool {
	if this.lastPage != nil {
		return true
	}
	if this.skipToEnd {
		os.Stdout.WriteString(msgWithSkip)
	} else {
		os.Stdout.WriteString(msg)
	}

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
			case 'K':
				fallthrough
			case 'k':
				rv = true
				this.lastPage = make([][]byte, this.height+2)
				this.lpStart = 0
				this.lpNext = 0
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
	if this.lastPage != nil {
		return this.recordForLP(buf)
	}
	for i := range this.w {
		_, err := this.w[i].w.Write(buf)
		if err != nil {
			return 0, err
		}
	}
	return len(buf), nil
}

const (
	_UTF8_MARKER = 0xc0
)

func runeDisplaySize(r rune) int {
	switch {
	case r < 0x100: // generally safe
		return 1
	default:
		return runewidth.RuneWidth(r)
	}
}

func (this *Pager) Write(buf []byte) (int, error) {
	if len(this.w) == 0 {
		return 0, io.EOF
	}
	if !this.paging || this.skip || len(this.w) == 1 && this.w[0].w != os.Stdout {
		return this.copyToOutput(buf)
	}
	start := 0
	for n := 0; n < len(buf); n++ {
		if this.skip {
			sn, err := this.copyToOutput(buf[n:])
			if err == nil {
				n += sn
			}
			return n, err
		}
		if buf[n] == 0x4 { // EOT
			_, err := this.copyToOutput(buf[start:n])
			if err != nil {
				return 0, err
			}
			err = this.writeLastPage()
			if err != nil {
				return 0, err
			}
			return n, nil
		} else if buf[n] == '\n' {
			_, err := this.copyToOutput(buf[start : n+1])
			if err != nil {
				return 0, err
			}
			start = n + 1
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
			if buf[n]&_UTF8_MARKER == _UTF8_MARKER {
				r, sz := utf8.DecodeRune(buf[n:])
				if r == utf8.RuneError && (sz == 1 || sz == 0) {
					return 0, io.EOF
				} else {
					n += sz - 1
					w := runeDisplaySize(r)
					this.col += w
				}
			} else {
				this.col++
			}
			if n+1 >= len(buf) || buf[n+1] != '\n' {
				if this.col >= int(this.width) {
					_, err := this.copyToOutput(buf[start : n+1])
					if err != nil {
						return 0, err
					}
					start = n + 1
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
	}
	if start < len(buf) {
		_, err := this.copyToOutput(buf[start:])
		if err != nil {
			return 0, err
		}
		this.col += len(buf[start:])
		this.lpCont = true
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

func (this *Pager) recordForLP(buf []byte) (int, error) {
	if this.lpCont {
		n := this.lpNext - 1
		if n < 0 {
			n = len(this.lastPage) - 1
		}
		this.lastPage[n] = append(this.lastPage[n], buf...)
		this.lpCont = false
		return len(buf), nil
	}
	// copy as the buffer given to us may be reused by the caller
	if this.lastPage[this.lpNext] == nil || cap(this.lastPage[this.lpNext]) < len(buf) {
		this.lastPage[this.lpNext] = make([]byte, len(buf))
	}
	this.lastPage[this.lpNext] = this.lastPage[this.lpNext][:len(buf)]
	copy(this.lastPage[this.lpNext], buf)
	this.lpNext++
	if this.lpNext == len(this.lastPage) {
		this.lpNext = 0
	}
	if this.lpNext == this.lpStart {
		this.lpStart++
		if this.lpStart == len(this.lastPage) {
			this.lpStart = 0
		}
	}
	return len(buf), nil
}

func (this *Pager) writeLastPage() error {
	if this.lastPage != nil {
		skipNL := 0
		if !this.lpCont {
			skipNL++
		}
		for i := range this.w {
			_, err := this.w[i].w.Write([]byte("\n-- skip to end --\n")[skipNL:])
			if err != nil {
				return err
			}
			n := 0
			for b := this.lpStart; b != this.lpNext; {
				_, err := this.w[i].w.Write(this.lastPage[b])
				if err != nil {
					return err
				}
				b++
				if b == len(this.lastPage) {
					b = 0
				}
				n++
			}
		}
	}
	return nil
}
