//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build linux || darwin
// +build linux darwin

package vliner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	pliner "github.com/peterh/liner"
)

func NewLiner() *State {
	var s = State{}

	s.r = bufio.NewReader(os.Stdin)

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), _TCGETS, uintptr(unsafe.Pointer(&s.origMode)))
	if 0 != errno {
		s.pipeIn = true
	} else {
		s.newMode = s.origMode
		s.newMode.Iflag &^= (syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR |
			syscall.IGNCR | syscall.ICRNL | syscall.IXON)
		s.newMode.Iflag |= (syscall.BRKINT | syscall.IGNPAR | syscall.PARMRK)
		// we have to handle job control (SIGSTOP generation) directly so we can restore ECHO before the shell resumes
		// as some shells (notably ksh) has issues when this isn't done, hence resetting ISIG
		s.newMode.Lflag &^= (syscall.ISIG | syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.IEXTEN)
		s.isig = true
		s.newMode.Cflag &^= (syscall.CSIZE | syscall.PARENB)
		s.newMode.Cflag |= syscall.CS8
		s.newMode.Cc[syscall.VTIME] = 2 // read will block for 200ms
		s.newMode.Cc[syscall.VMIN] = 0  // may return nothing

		s.controlChars[ccVINTR] = rune(s.origMode.Cc[syscall.VINTR])
		s.controlChars[ccVEOF] = rune(s.origMode.Cc[syscall.VEOF])
		s.controlChars[ccVLNEXT] = rune(s.origMode.Cc[syscall.VLNEXT])
		s.controlChars[ccVERASE] = rune(s.origMode.Cc[syscall.VERASE])
		s.controlChars[ccVWERASE] = rune(s.origMode.Cc[syscall.VWERASE])
		s.controlChars[ccVKILL] = rune(s.origMode.Cc[syscall.VKILL])
		s.controlChars[ccVSUSP] = rune(s.origMode.Cc[syscall.VSUSP])
		s.controlChars[ccDigraph] = rune(_ASCII_VT) // Ctrl+K

	}

	ierrno := s.getWinSize()
	if 0 != ierrno {
		s.pipeOut = true
	} else if !s.pipeIn {
		s.signals = make(chan os.Signal, 1)
		signal.Notify(s.signals, syscall.SIGWINCH, syscall.SIGCONT, syscall.SIGINT)
	}

	s.history = make([]string, 0, pliner.HistoryLimit)
	s.cmdRepeat = make([]rune, 0, 64)
	s.buffer = make([]rune, 0, 1024)

	return &s
}

func (s *State) startTerm() error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), _TCSETS, uintptr(unsafe.Pointer(&s.newMode)))
	if 0 != errno {
		return errno
	}
	s.needsReset = true
	return nil
}

func (s *State) resetTerm() error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), _TCSETS, uintptr(unsafe.Pointer(&s.origMode)))
	if 0 != errno {
		return errno
	}
	s.needsReset = false
	return nil
}

func (s *State) setISig(on bool) {
	s.isig = on
}

func (s *State) getWinSize() int {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&s.ws)))
	return int(errno)
}

func (s *State) moveUp(n int) {
	if 0 < n {
		if n > s.cy {
			n = s.cy
		}
		fmt.Printf(fmt.Sprintf("\033[%dA", n))
		s.cy -= n
	}
}

func (s *State) moveDown(n int) {
	s.cy += n
	if s.promptLines <= s.cy {
		for ; n > 0; n-- {
			fmt.Printf("\033D")
		}
		s.promptLines = s.cy + 1
	} else if 0 < n {
		fmt.Printf(fmt.Sprintf("\033[%dB", n))
	}
}

func (s *State) moveRight(n int) {
	if 0 < n {
		s.cx += n
		for int(s.ws.Col) < s.cx {
			s.cx -= int(s.ws.Col)
			s.cy++
		}
		fmt.Printf(fmt.Sprintf("\033[%dC", n))
	}
}

func (s *State) moveToCol(n int) {
	if int(s.ws.Col) <= n {
		n = int(s.ws.Col) - 1
	}
	s.cx = 0
	fmt.Printf("\r")
	s.moveRight(n)
}

func (s *State) moveToStart() {
	s.cx = 0
	fmt.Printf("\r")
	s.moveUp(s.cy)
}

func (s *State) clearToEOL() {
	fmt.Printf("\033[K")
}

func (s *State) clearLine() {
	fmt.Printf("\r\033[K")
}

func (s *State) clearToEOP() {
	fmt.Printf("\033[K")
	for i := s.cy + 1; s.promptLines > i; i++ {
		s.moveDown(1)
		s.clearLine()
	}
}

func (s *State) clearPrompt() {
	s.moveToStart()
	s.clearToEOP()
	s.moveToStart()
}

func writeStrNoWrapInternal(str []rune, trunc bool) bool {
	fmt.Printf("\033[?7l%s\033[?7h", string(str))
	return trunc
}

func (s *State) read() (rune, error) {
	var r rune
	var err error
	// first check if there are any characters queued up for us to process ahead of further live input
	if 0 < len(s.buffer) {
		r = s.buffer[0]
		s.buffer = s.buffer[1:]
		return r, nil
	}
	for {
		select {
		case sig := <-s.signals:
			if syscall.SIGINT == sig {
				return s.controlChars[ccVINTR], nil
			}
			if syscall.SIGCONT == sig {
				s.startTerm()
			}
			s.getWinSize()
			return _REPLAY_END, nil
		default:
			var sz int
			r, sz, err = s.r.ReadRune()
			if io.EOF != err && 0 < sz {
				if s.isig && s.controlChars[ccVSUSP] == r {
					s.resetTerm()
					p, _ := os.FindProcess(os.Getpid())
					p.Signal(syscall.SIGSTOP)
					// make sure there is time for the signal to be delivered before we continue
					time.Sleep(100 * time.Millisecond)
				} else {
					return r, err
				}
			}
		}
	}
}

func (s *State) hideCursor() {
	fmt.Printf("\033[?25l")
}

func (s *State) showCursor() {
	fmt.Printf("\033[?25h")
}
