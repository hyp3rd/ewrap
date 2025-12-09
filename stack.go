package ewrap

import (
	"runtime"
	"strings"
)

// StackFrame represents a single frame in a stack trace.
type StackFrame struct {
	// Function is the fully qualified function name
	Function string `json:"function" yaml:"function"`
	// File is the source file path
	File string `json:"file"     yaml:"file"`
	// Line is the line number in the source file
	Line int `json:"line"     yaml:"line"`
	// PC is the program counter for this frame
	PC uintptr `json:"pc"       yaml:"pc"`
}

// StackTrace represents a collection of stack frames.
type StackTrace []StackFrame

// StackIterator provides a way to iterate through stack frames.
type StackIterator struct {
	frames []StackFrame
	index  int
}

// NewStackIterator creates a new stack iterator from program counters.
func NewStackIterator(pcs []uintptr) *StackIterator {
	frames := make([]StackFrame, 0, len(pcs))
	callersFrames := runtime.CallersFrames(pcs)

	for {
		frame, more := callersFrames.Next()

		// Skip runtime frames and error package frames
		if !strings.Contains(frame.File, "runtime/") &&
			!strings.Contains(frame.File, "ewrap/errors.go") {
			frames = append(frames, StackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
				PC:       frame.PC,
			})
		}

		if !more {
			break
		}
	}

	return &StackIterator{
		frames: frames,
		index:  0,
	}
}

// Next returns the next stack frame, or nil if no more frames.
func (si *StackIterator) Next() *StackFrame {
	if si.index >= len(si.frames) {
		return nil
	}

	frame := &si.frames[si.index]
	si.index++

	return frame
}

// HasNext returns true if there are more frames to iterate.
func (si *StackIterator) HasNext() bool {
	return si.index < len(si.frames)
}

// Reset resets the iterator to the beginning.
func (si *StackIterator) Reset() {
	si.index = 0
}

// Frames returns all remaining frames as a slice.
func (si *StackIterator) Frames() []StackFrame {
	if si.index >= len(si.frames) {
		return nil
	}

	return si.frames[si.index:]
}

// AllFrames returns all frames regardless of current position.
func (si *StackIterator) AllFrames() []StackFrame {
	return si.frames
}

// GetStackIterator returns a stack iterator for the error's stack trace.
func (e *Error) GetStackIterator() *StackIterator {
	return NewStackIterator(e.stack)
}

// GetStackFrames returns all stack frames as a slice.
func (e *Error) GetStackFrames() []StackFrame {
	iterator := e.GetStackIterator()

	return iterator.AllFrames()
}
