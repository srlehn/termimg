// <Copyright> 2018-2023 Simon Robin Lehn. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package qdefault

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

const (
	timeOutIntervalDefault = 50 * time.Millisecond
	timeOutMaxDefault      = 1000 * time.Millisecond
	closeWaitDefault       = 200 * time.Millisecond
)

var _ term.Querier = (*querierDefault)(nil)
var _ term.CachedQuerier = (*querierDefault)(nil)

func NewQuerier() term.Querier { return &querierDefault{} }

type querierDefault struct {
	muQuery         sync.Mutex
	r               chan rune
	drain           atomic.Value // bool
	prevQueryFailed atomic.Value // bool
	logger          *log.Logger
	timeOutInterval time.Duration
	timeOutMax      time.Duration
	closeWait       time.Duration
}

// Close ...
func (q *querierDefault) Close() error {
	if q == nil {
		return nil
	}
	q.r = nil
	drain, okDrain := q.drain.Load().(bool)
	fail, okFail := q.prevQueryFailed.Load().(bool)
	if (!okDrain || !drain) || (!okFail || fail) {
		if q.closeWait == 0 {
			q.closeWait = closeWaitDefault
		}
		time.Sleep(q.closeWait)
	}
	return nil
}

func (q *querierDefault) startReading(tty term.TTY) {
	if q == nil {
		panic(`nil receiver`)
	}
	if q.r != nil {
		return
	}
	q.r = make(chan rune)

	go func() {
		var (
			rDr           rune
			errDr         error
			drain         bool
			drainLeftOver bool
		)

	read:
		for {
			var r rune
			var err error
			if !drain && drainLeftOver {
				r, err = rDr, errDr
			} else {
				r, _, err = tty.ReadRune() // TODO use io.Reader
			}
			if dr, ok := q.drain.Load().(bool); ok {
				drain = dr
			}
			if drain {
				rDr, errDr = r, err
				drainLeftOver = true
			}
			if !drain && err != nil && q.logger != nil {
				q.logger.Println(err)
			}
			if q.r == nil { // TODO guard this?
				break read
			}
			if !drain {
				select {
				case q.r <- r:
				case <-time.After(q.timeOutInterval):
				}
			}
		}
	}()
}

// Query ...
func (q *querierDefault) Query(qs string, tty term.TTY, p term.Parser) (string, error) {
	if q == nil {
		return ``, errors.New(consts.ErrNilReceiver)
	}
	q.muQuery.Lock()
	defer q.muQuery.Unlock()
	q.startReading(tty)
	q.drain.Store(false)
	defer func() { q.drain.Store(true) }()
	tty.Write([]byte(qs))
	var (
		repl        string
		errRead     error
		readStop    bool
		time1       time.Time
		readChan    = make(chan struct{}, 1)
		timeChan    = make(chan struct{})
		timeoutChan = make(chan struct{})
	)

	if q.timeOutMax == 0 {
		q.timeOutMax = timeOutMaxDefault
	}
	if q.timeOutInterval == 0 {
		// mlterm: avg: ~30µs, spikes: 1.5ms
		// VTE-based terminals seem to take much longer
		q.timeOutInterval = timeOutIntervalDefault
	}

	// stop reading if interval between single reads is too large
	go func() {
		for range timeChan {
			time1 = time.Now()
			go func(time2 time.Time) {
				<-time.After(q.timeOutInterval)
				if time1 == time2 { // time1 was not updated in q.timeOutInterval
					timeoutChan <- struct{}{}
				}
			}(time1)
		}
	}()
	defer func() { timeChan = nil }()

	readChan <- struct{}{}
read:
	for {
		select {
		case <-readChan:
			timeChan <- struct{}{}
			if errRead != nil {
				return ``, errRead
			}
			if readStop {
				break read
			}
			go func() {
				r, ok := <-q.r
				if !ok {
					q.r = nil
					errRead = errors.New(`tty read chan closed`)
					return
				}
				timeChan <- struct{}{}
				repl += string(r)
				readStop = p != nil && p.Parse(r)
				readChan <- struct{}{}
			}()
		case <-time.After(q.timeOutMax):
			q.prevQueryFailed.Store(true)
			return ``, errors.New("time out")
		case <-timeoutChan:
			if p == nil {
				break read
			} else {
				q.prevQueryFailed.Store(true)
				return ``, errors.New(consts.ErrTimeoutInterval)
			}
		}
	}

	q.prevQueryFailed.Store(p == nil)

	return repl, nil
}

// CachedQuery ...
func (q *querierDefault) CachedQuery(qs string, tty term.TTY, p term.Parser, pr environ.Proprietor) (string, error) {
	if q == nil {
		return ``, errors.New(consts.ErrNilReceiver)
	}
	return term.CachedQuery(q, qs, tty, p, pr, pr)
}
