//go:build !windows
// +build !windows

package uv

import (
	"context"
)

// streamData sends data from the input stream to the event channel.
func (p *TerminalReader) streamData(ctx context.Context, readc chan []byte) error {
	return p.sendBytes(ctx, readc)
}
