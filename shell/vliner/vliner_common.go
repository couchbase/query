//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build linux || darwin

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

func NewLiner() (*State, error) {
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
		s.controlChars[ccUP] = rune(-'A')
		s.controlChars[ccDOWN] = rune(-'B')
		s.controlChars[ccRIGHT] = rune(-'C')
		s.controlChars[ccLEFT] = rune(-'D')

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

	return &s, nil
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

func writeStrNoWrapInternal(str []rune) {
	fmt.Printf("\033[?7l%s\033[?7h", string(str))
}

func writeAtEOL(ch rune) {
	fmt.Printf("%c", ch)
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
				} else if nil == err && _ASCII_ESC == r {
					p, e := s.r.Peek(2)
					if e == nil && '[' == p[0] {
						skip := false
						switch p[1] {
						case 'A':
							r = s.controlChars[ccUP]
						case 'B':
							r = s.controlChars[ccDOWN]
						case 'C':
							r = s.controlChars[ccRIGHT]
						case 'D':
							r = s.controlChars[ccLEFT]
						default:
							skip = true
						}
						if !skip {
							s.r.ReadRune()
							s.r.ReadRune()
						}
					}
					return r, err
				} else {
					return r, err
				}
			}
		}
	}
}
