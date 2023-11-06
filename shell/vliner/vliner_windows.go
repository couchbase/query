//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build windows

package vliner

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	pliner "github.com/peterh/liner"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	winReadConsoleInput           = kernel32.NewProc("ReadConsoleInputW")
	winPeekConsoleInput           = kernel32.NewProc("PeekConsoleInputW")
	winSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	winGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	winFillConsoleOutputCharacter = kernel32.NewProc("FillConsoleOutputCharacterW")
)

type Termios struct {
	imode uint32
	omode uint32
}

type coord struct {
	x int16
	y int16
}

type smallRect struct {
	left   int16
	top    int16
	right  int16
	bottom int16
}

type consoleScreenBufferInfo struct {
	dwSize              coord
	dwCursorPosition    coord
	wAttributes         int16
	srWindow            smallRect
	dwMaximumWindowSize coord
}

type consoleKeyEvent struct {
	eventType uint16
	pad       uint16
	pressed   int32
	repeat    uint16
	keyCode   uint16
	scanCode  uint16
	chr       uint16
	modifiers uint32
}

type consoleCursorInfo struct {
	size    uint32
	visible uint32
}

const (
	_ENABLE_PROCESSED_INPUT        = uint32(0x0001)
	_ENABLE_LINE_INPUT             = uint32(0x0002)
	_ENABLE_ECHO_INPUT             = uint32(0x0004)
	_ENABLE_WINDOW_INPUT           = uint32(0x0008)
	_ENABLE_MOUSE_INPUT            = uint32(0x0010)
	_ENABLE_INSERT_MODE            = uint32(0x0020)
	_ENABLE_VIRTUAL_TERMINAL_INPUT = uint32(0x0200)

	_ENABLE_PROCESSED_OUTPUT            = uint32(0x0001)
	_ENABLE_WRAP_AT_EOL_OUTPUT          = uint32(0x0002)
	_ENABLE_VIRTUAL_TERMINAL_PROCESSING = uint32(0x0004)
	_DISABLE_NEWLINE_AUTO_RETURN        = uint32(0x0008)

	_WINDOW_BUFFER_SIZE_EVENT = 4
	_KEY_EVENT                = 1
)

func NewLiner() (*State, error) {
	var s = State{}

	err := syscall.GetConsoleMode(syscall.Stdin, &s.origMode.imode)
	if nil != err {
		if err.Error() == "The handle is invalid." {
			err = fmt.Errorf("No console access.")
		}
		return nil, err
	}
	err = syscall.GetConsoleMode(syscall.Stdout, &s.origMode.omode)
	if nil != err {
		return nil, err
	}

	s.newMode = s.origMode

	s.newMode.imode &^= _ENABLE_PROCESSED_INPUT | _ENABLE_LINE_INPUT | _ENABLE_ECHO_INPUT | _ENABLE_MOUSE_INPUT |
		_ENABLE_INSERT_MODE
	s.newMode.imode |= _ENABLE_WINDOW_INPUT | _ENABLE_VIRTUAL_TERMINAL_INPUT

	s.newMode.omode |= _ENABLE_PROCESSED_OUTPUT | _ENABLE_WRAP_AT_EOL_OUTPUT |
		_ENABLE_VIRTUAL_TERMINAL_PROCESSING | _DISABLE_NEWLINE_AUTO_RETURN

	s.controlChars[ccVINTR] = rune(0x3)         // Ctrl+C
	s.controlChars[ccVEOF] = rune(0x4)          // Ctrl+D
	s.controlChars[ccVLNEXT] = rune(0x16)       // Ctrl+V
	s.controlChars[ccVERASE] = rune(0x7f)       // DEL
	s.controlChars[ccVWERASE] = rune(0x17)      // Ctrl+W
	s.controlChars[ccVKILL] = rune(0x1c)        // Ctrl+\
	s.controlChars[ccVSUSP] = rune(0x1a)        // Ctrl+Z
	s.controlChars[ccDigraph] = rune(_ASCII_VT) // Ctrl+K
	s.controlChars[ccUP] = rune(-'A')
	s.controlChars[ccDOWN] = rune(-'B')
	s.controlChars[ccRIGHT] = rune(-'C')
	s.controlChars[ccLEFT] = rune(-'D')

	s.getWinSize()

	s.history = make([]string, 0, pliner.HistoryLimit)
	s.cmdRepeat = make([]rune, 0, 64)
	s.buffer = make([]rune, 0, 10240)

	return &s, nil
}

func (s *State) startTerm() error {
	success, _, err := winSetConsoleMode.Call(uintptr(syscall.Stdin), uintptr(s.newMode.imode))
	if 1 != success {
		return err
	}
	success, _, err = winSetConsoleMode.Call(uintptr(syscall.Stdout), uintptr(s.newMode.omode))
	if 1 != success {
		return err
	}
	s.needsReset = true
	fmt.Printf("\033[!p")
	return nil
}

func (s *State) resetTerm() error {
	fmt.Printf("\033[!p")
	success, _, err := winSetConsoleMode.Call(uintptr(syscall.Stdin), uintptr(s.origMode.imode))
	if 1 != success {
		return err
	}
	success, _, err = winSetConsoleMode.Call(uintptr(syscall.Stdout), uintptr(s.origMode.omode))
	if 1 != success {
		return err
	}
	s.needsReset = false
	return nil
}

func (s *State) setISig(on bool) {
}

func (s *State) getWinSize() int {
	var sbi consoleScreenBufferInfo
	winGetConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&sbi)))
	s.ws.Row = uint16(sbi.srWindow.bottom-sbi.srWindow.top) + 1
	s.ws.Col = uint16(sbi.srWindow.right-sbi.srWindow.left) + 1
	return 0
}

func writeStrNoWrapInternal(str []rune) {
	fmt.Printf("%s", string(str))
}

func writeAtEOL(ch rune) {
	var sbi consoleScreenBufferInfo
	var nr uint32
	winGetConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&sbi)))
	coord := int(sbi.dwCursorPosition.x) | (int(sbi.dwCursorPosition.y) << 16)
	winFillConsoleOutputCharacter.Call(uintptr(syscall.Stdout), uintptr(ch), uintptr(1), uintptr(coord),
		uintptr(unsafe.Pointer(&nr)))
}

func (s *State) read() (rune, error) {
	var ev [3]consoleKeyEvent
	var rv uint32
	pev := uintptr(unsafe.Pointer(&ev[0]))
	// first check if there are any characters queued up for us to process ahead of further live input
	if 0 < len(s.buffer) {
		r := s.buffer[0]
		s.buffer = s.buffer[1:]
		return r, nil
	}
	for {
		success, _, err := winReadConsoleInput.Call(uintptr(syscall.Stdin), pev, 1, uintptr(unsafe.Pointer(&rv)))
		if 1 != success || 1 != rv {
			return rune(0), err
		}
		switch ev[0].eventType {
		case _WINDOW_BUFFER_SIZE_EVENT:
			s.getWinSize()
			return _REPLAY_END, nil
		case _KEY_EVENT:
			if 0 != ev[0].pressed {
				if _ASCII_ESC == ev[0].chr {
					pev = uintptr(unsafe.Pointer(&ev[1]))
					success, _, _ := winPeekConsoleInput.Call(uintptr(syscall.Stdin), pev, 2, uintptr(unsafe.Pointer(&rv)))
					r := rune(ev[0].chr)
					if 1 == success && 2 == rv && '[' == ev[1].chr {
						skip := false
						switch ev[2].chr {
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
							winReadConsoleInput.Call(uintptr(syscall.Stdin), pev, 2, uintptr(unsafe.Pointer(&rv)))
						}
					}
					return r, nil
				} else if 0 != ev[0].chr {
					return rune(ev[0].chr), nil
				}
			}
		}
	}
}

func invokeEditor(args []string, attr *syscall.ProcAttr) bool {
	if nil == attr.Sys {
		attr.Sys = &syscall.SysProcAttr{}
	}
	attr.Sys.CreationFlags |= 0x10 // CREATE_NEW_CONSOLE
	_, handle, err := syscall.StartProcess(args[0], args, attr)
	if nil == err {
		_, err = syscall.WaitForSingleObject(syscall.Handle(handle), syscall.INFINITE)
	}
	return nil == err
}

func setupPipe(cmd string) *exec.Cmd {
	return exec.Command("cmd.exe", "/c", cmd)
}
