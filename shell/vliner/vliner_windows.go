//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build windows
// +build windows

package vliner

import (
	"fmt"
	"syscall"
	"unsafe"

	pliner "github.com/peterh/liner"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	winReadConsoleInput           = kernel32.NewProc("ReadConsoleInputW")
	winSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	winSetConsoleCursorPosition   = kernel32.NewProc("SetConsoleCursorPosition")
	winGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	winFillConsoleOutputCharacter = kernel32.NewProc("FillConsoleOutputCharacterW")
	winGetConsoleCursorInfo       = kernel32.NewProc("GetConsoleCursorInfo")
	winSetConsoleCursorInfo       = kernel32.NewProc("SetConsoleCursorInfo")
)

type Termios struct {
	mode uint32
}

type coord struct {
	x uint16
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

func NewLiner() *State {
	var s = State{}

	syscall.GetConsoleMode(syscall.Stdin, &s.origMode.mode)

	s.newMode = s.origMode
	s.newMode.mode &^= 0x0001 | 0x0002 | 0x0004 | 0x0010 | 0x0020
	s.newMode.mode |= 0x0008

	s.controlChars[ccVINTR] = rune(0x3)         // Ctrl+C
	s.controlChars[ccVEOF] = rune(0x4)          // Ctrl+D
	s.controlChars[ccVLNEXT] = rune(0x16)       // Ctrl+V
	s.controlChars[ccVERASE] = rune(0x8)        // backspace
	s.controlChars[ccVWERASE] = rune(0x17)      // Ctrl+W
	s.controlChars[ccVKILL] = rune(0x1c)        // Ctrl+\
	s.controlChars[ccVSUSP] = rune(0x1a)        // Ctrl+Z
	s.controlChars[ccDigraph] = rune(_ASCII_VT) // Ctrl+K

	s.getWinSize()

	s.history = make([]string, 0, pliner.HistoryLimit)
	s.cmdRepeat = make([]rune, 0, 64)
	s.buffer = make([]rune, 0, 1024)

	return &s
}

func (s *State) startTerm() error {
	success, _, err := winSetConsoleMode.Call(uintptr(syscall.Stdin), uintptr(s.newMode.mode))
	if 1 != success {
		return err
	}
	s.needsReset = true
	return nil
}

func (s *State) resetTerm() error {
	success, _, err := winSetConsoleMode.Call(uintptr(syscall.Stdin), uintptr(s.origMode.mode))
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
	s.ws.Row = uint16(sbi.dwSize.y)
	s.ws.Col = uint16(sbi.dwSize.x)
	return 0
}

func (s *State) moveUp(n int) {
	if 0 < n {
		if n > s.cy {
			n = s.cy
		}
		var sbi consoleScreenBufferInfo
		winGetConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&sbi)))
		sbi.dwCursorPosition.y -= int16(n)
		ncp := int(sbi.dwCursorPosition.x) | (int(sbi.dwCursorPosition.y) << 16)
		winSetConsoleCursorPosition.Call(uintptr(syscall.Stdout), uintptr(ncp))
		s.cy -= n
	}
}

func (s *State) moveDown(n int) {
	if 0 < n {
		s.cy += n
		if s.promptLines <= s.cy {
			s.promptLines = s.cy + 1
		}
		var sbi consoleScreenBufferInfo
		winGetConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&sbi)))
		sbi.dwCursorPosition.y += int16(n)
		ncp := int(sbi.dwCursorPosition.x) | (int(sbi.dwCursorPosition.y) << 16)
		winSetConsoleCursorPosition.Call(uintptr(syscall.Stdout), uintptr(ncp))
	}
}

func (s *State) moveRight(n int) {
	if 0 < n {
		s.cx += n
		y := int16(0)
		for int(s.ws.Col) < s.cx {
			s.cx -= int(s.ws.Col)
			s.cy++
			y++
		}
		var sbi consoleScreenBufferInfo
		winGetConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&sbi)))
		sbi.dwCursorPosition.y += y
		sbi.dwCursorPosition.x += uint16(s.cx)
		ncp := int(sbi.dwCursorPosition.x) | (int(sbi.dwCursorPosition.y) << 16)
		winSetConsoleCursorPosition.Call(uintptr(syscall.Stdout), uintptr(ncp))
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
	var sbi consoleScreenBufferInfo
	var nr uint32
	winGetConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&sbi)))
	coord := int(sbi.dwCursorPosition.x) | (int(sbi.dwCursorPosition.y) << 16)
	l := sbi.dwSize.x - sbi.dwCursorPosition.x
	winFillConsoleOutputCharacter.Call(uintptr(syscall.Stdout), uintptr(' '), uintptr(l), uintptr(coord),
		uintptr(unsafe.Pointer(&nr)))
}

func (s *State) clearLine() {
	fmt.Printf("\r")
	var sbi consoleScreenBufferInfo
	var nr uint32
	winGetConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&sbi)))
	coord := 0 | (int(sbi.dwCursorPosition.y) << 16)
	l := sbi.dwSize.x
	winFillConsoleOutputCharacter.Call(uintptr(syscall.Stdout), uintptr(' '), uintptr(l), uintptr(coord),
		uintptr(unsafe.Pointer(&nr)))
}

func (s *State) clearToEOP() {
	s.clearToEOL()
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
	if !trunc {
		fmt.Printf("%s", string(str))
	} else {
		fmt.Printf("%s", string(str[:len(str)-1]))
		ch := str[len(str)-1]
		var sbi consoleScreenBufferInfo
		var nr uint32
		winGetConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&sbi)))
		coord := int(sbi.dwCursorPosition.x) | (int(sbi.dwCursorPosition.y) << 16)
		winFillConsoleOutputCharacter.Call(uintptr(syscall.Stdout), uintptr(ch), uintptr(1), uintptr(coord),
			uintptr(unsafe.Pointer(&nr)))
	}
	return trunc
}

func (s *State) read() (rune, error) {
	var ev consoleKeyEvent
	var rv uint32
	pev := uintptr(unsafe.Pointer(&ev))
	// first check if there are any characters queued up for us to process ahead of further live input
	if 0 < len(s.buffer) {
		r := s.buffer[0]
		s.buffer = s.buffer[1:]
		return r, nil
	}
	for {
		success, _, err := winReadConsoleInput.Call(uintptr(syscall.Stdin), pev, 1, uintptr(unsafe.Pointer(&rv)))
		if 1 != success || 1 != rv {
			return 0, err
		}
		switch ev.eventType {
		case 4: // window size
			s.getWinSize()
			return _REPLAY_END, nil
		case 1: // key
			if 0 != ev.pressed {
				if 0 != ev.chr {
					return rune(ev.chr), nil
				}
			}
		}
	}
}

func (s *State) hideCursor() {
	var cci consoleCursorInfo
	winGetConsoleCursorInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&cci)))
	cci.visible = 0
	winSetConsoleCursorInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&cci)))
}

func (s *State) showCursor() {
	var cci consoleCursorInfo
	winGetConsoleCursorInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&cci)))
	cci.visible = 1
	winSetConsoleCursorInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&cci)))
}
