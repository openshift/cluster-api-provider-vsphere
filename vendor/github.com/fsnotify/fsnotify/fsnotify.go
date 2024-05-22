//go:build !plan9
// +build !plan9

// Package fsnotify provides a cross-platform interface for file system
// notifications.
package fsnotify

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// Event represents a single file system notification.
type Event struct {
	Name string // Relative path to the file or directory.
	Op   Op     // File operation that triggered the event.
}

// Op describes a set of file operations.
type Op uint32

// These are the generalized file operations that can trigger a notification.
const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

// Common errors that can be reported by a watcher
var (
	ErrNonExistentWatch = errors.New("can't remove non-existent watcher")
	ErrEventOverflow    = errors.New("fsnotify queue overflow")
)

func (op Op) String() string {
	var b strings.Builder
	if op.Has(Create) {
		b.WriteString("|CREATE")
	}
	if op.Has(Remove) {
		b.WriteString("|REMOVE")
	}
	if op.Has(Write) {
		b.WriteString("|WRITE")
	}
	if op.Has(Rename) {
		b.WriteString("|RENAME")
	}
	if op.Has(Chmod) {
		b.WriteString("|CHMOD")
	}
	if buffer.Len() == 0 {
		return ""
	}
	return buffer.String()[1:] // Strip leading pipe
}

// Has reports if this operation has the given operation.
func (o Op) Has(h Op) bool { return o&h == h }

// Has reports if this event has the given operation.
func (e Event) Has(op Op) bool { return e.Op.Has(op) }

// String returns a string representation of the event with their path.
func (e Event) String() string {
	return fmt.Sprintf("%q: %s", e.Name, e.Op.String())
}
