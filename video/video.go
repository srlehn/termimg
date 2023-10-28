//go:build dev

package video

import (
	"github.com/gopxl/beep"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type MediaPlayer struct {
	canvas *term.Canvas
	audio  beep.Streamer
	player func(beep.Streamer) error
}

func NewMediaPlayer(canvas *term.Canvas, audio beep.Streamer, player func(beep.Streamer) error) (*MediaPlayer, error) {
	if err := errors.NilParam(canvas, audio, player); err != nil {
		return nil, err
	}
	return &MediaPlayer{
		canvas: canvas,
		audio:  audio,
		player: player,
	}, nil
}

func (a *MediaPlayer) Pause() error {
	// TODO send SIGTSTP/SIGSTOP to ffmpeg
	return errors.NotImplemented()
}

func (a *MediaPlayer) Continue() error {
	// TODO send SIGCONT to ffmpeg
	return errors.NotImplemented()
}
