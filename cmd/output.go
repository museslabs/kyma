package cmd

import (
	"os"
	"sync"

	"github.com/charmbracelet/x/ansi"
)

// Sequences that begin and end a synchronized update, during which the
// terminal holds off drawing until the whole of it has arrived.
var (
	beginUpdate = ansi.SetMode(ansi.ModeSynchronizedOutput)
	endUpdate   = ansi.ResetMode(ansi.ModeSynchronizedOutput)
)

// syncedOutput is the terminal, written to in synchronized updates. The
// renderer writes each frame out in one go, so wrapping every write makes the
// terminal show whole frames only, and never draw one that is still arriving,
// which tears an animation across the screen. Terminals that don't know
// synchronized updates ignore them.
type syncedOutput struct {
	*os.File

	mu  sync.Mutex
	buf []byte
}

func newSyncedOutput(f *os.File) *syncedOutput {
	return &syncedOutput{File: f}
}

func (o *syncedOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// One write for the whole update, so nothing else written to the
	// terminal can land in the middle of it.
	o.buf = append(o.buf[:0], beginUpdate...)
	o.buf = append(o.buf, p...)
	o.buf = append(o.buf, endUpdate...)
	if _, err := o.File.Write(o.buf); err != nil {
		return 0, err
	}
	return len(p), nil
}
