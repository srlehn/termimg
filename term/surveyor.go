package term

import (
	"fmt"
	"image"
	"sync"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/propkeys"
)

type Resolution struct {
	TermInCellsW, TermInCellsH uint
	TermInPxlsW, TermInPxlsH   uint
	CellInPxlsW, CellInPxlsH   float64
}

type SurveyorLight interface {
	CellSize(TTY, Querier, Window, Properties) (width, height float64, err error)
	SizeInCells(TTY, Querier, Window, Properties) (width, height uint, err error)
	SizeInPixels(TTY, Querier, Window, Properties) (width, height uint, err error)
	Cursor(TTY, Querier, Window, Properties) (xPosCells, yPosCells uint, err error)
	SetCursor(xPosCells, yPosCells uint, tty TTY, qu Querier, w Window, pr Properties) (err error)
	WatchResizeEventsStart(TTY, Querier, Window, Properties) (_ <-chan Resolution, closeFunc func() error, _ error)
	WatchResizeEventsStop() error
}

// Surveyor is implemented by Terminal
type Surveyor interface {
	// passes stored TTY, Querier, Window, Proprietor to a SurveyorLight
	CellSize() (width, height float64, err error)
	SizeInCells() (width, height uint, err error)
	SizeInPixels() (width, height uint, err error)
	Cursor() (xPosCells, yPosCells uint, err error)
	SetCursor(xPosCells, yPosCells uint) (err error)
	CellScale(ptSrcPx, ptDstCl image.Point) (ptSrcCl image.Point, _ error)
}

// PartialSurveyor implements some of:
//   - CellSize(tty TTY) (width, height float64, err error)
//   - CellSizeQuery(qu Querier, tty TTY) (width, height float64, err error)
//   - SizeInCells(tty TTY) (widthCells, heightCells uint, err error)
//   - SizeInCellsQuery(qu Querier, tty TTY) (widthCells, heightCells uint, err error)
//   - SizeInPixels(tty TTY) (widthPixels, heightPixels uint, err error)
//   - SizeInPixelsQuery(qu Querier, tty TTY) (widthPixels, heightPixels uint, err error)
//   - SizeInPixelsWindow(w Window) (widthPixels, heightPixels uint, err error)
//   - SizeInCellsAndPixels(tty TTY) (widthCells, heightCells, widthPixels, heightPixels uint, err error)
//   - Cursor(tty TTY) (xPosCells, yPosCells uint, err error)
//   - CursorQuery(qu Querier, tty TTY) (widthCells, heightCells uint, err error)
//   - SetCursor(xPosCells, yPosCells uint, tty TTY) (err error)
//   - SetCursorQuery(xPosCells, yPosCells uint, qu Querier, tty TTY) (err error)
//   - ResizeEvents(tty TTY) (<-chan Resolution, error)
//   - ResizeEventsWindow(w Window) (<-chan Resolution, error)
type PartialSurveyor interface {
	// TODO doc
	// TODO Window func
	IsPartialSurveyor()
}

var _ SurveyorLight = (*surveyor)(nil)

type surveyor struct {
	// TODO pass self as SurveyorLight to inner PartialSurveyor via stored funcs
	s                         PartialSurveyor
	avoidQuery                bool
	isRemote                  bool
	cellSizeFuncs             []func(TTY, Querier, Window) (width, height float64, _ error)
	SizeInCellsFuncs          []func(TTY, Querier, Window) (widthCells, heightCells uint, _ error)
	SizeInPixelsFuncs         []func(TTY, Querier, Window) (widthPixels, heightPixels uint, _ error)
	SizeInCellsAndPixelsFuncs []func(TTY, Querier, Window) (widthCells, heightCells, widthPixels, heightPixels uint, _ error)
	posGetFuncs               []func(TTY, Querier) (xPosCells, yPosCells uint, _ error)
	posSetFuncs               []func(xPosCells, yPosCells uint, tty TTY, qu Querier) error
	resizeEventFuncs          []func(TTY, Window) (_ <-chan Resolution, closeFunc func() error, _ error)
	watchWINCHStopFunc        func() error
}

func getSurveyor(s PartialSurveyor, p Properties) SurveyorLight {
	if s == nil || p == nil {
		return nil
	}
	_, avoidANSI := p.Property(propkeys.AvoidANSI)
	if !avoidANSI {
		if _, ok := s.(*SurveyorNoANSI); ok {
			avoidANSI = true
		}
	}
	_, isRemote := p.Property(propkeys.IsRemote)
	srv := &surveyor{
		s:          s,
		avoidQuery: avoidANSI,
		isRemote:   isRemote,
	}

	if cellSizer, ok := s.(interface {
		CellSize(tty TTY) (width, height float64, err error)
	}); ok {
		cellSizeFunc := func(tty TTY, _ Querier, _ Window) (width float64, height float64, err error) {
			return cellSizer.CellSize(tty)
		}
		srv.cellSizeFuncs = append(srv.cellSizeFuncs, cellSizeFunc)
	}
	if sizerInCells, ok := s.(interface {
		SizeInCells(tty TTY) (widthCells, heightCells uint, err error)
	}); ok {
		sizeInCellsFunc := func(tty TTY, _ Querier, _ Window) (widthCells uint, heightCells uint, err error) {
			return sizerInCells.SizeInCells(tty)
		}
		srv.SizeInCellsFuncs = append(srv.SizeInCellsFuncs, sizeInCellsFunc)
	}
	if sizerInPixels, ok := s.(interface {
		SizeInPixels(tty TTY) (widthPixels, heightPixels uint, err error)
	}); ok {
		sizeInPixelsFunc := func(tty TTY, _ Querier, _ Window) (widthPixels uint, heightPixels uint, err error) {
			return sizerInPixels.SizeInPixels(tty)
		}
		srv.SizeInPixelsFuncs = append(srv.SizeInPixelsFuncs, sizeInPixelsFunc)
	}
	if sizerInCellsAndPixels, ok := s.(interface {
		SizeInCellsAndPixels(tty TTY) (widthCells, heightCells, widthPixels, heightPixels uint, err error)
	}); ok {
		sizeInCellsAndPixelsFunc := func(tty TTY, _ Querier, _ Window) (widthCells uint, heightCells uint, widthPixels uint, heightPixels uint, err error) {
			return sizerInCellsAndPixels.SizeInCellsAndPixels(tty)
		}
		srv.SizeInCellsAndPixelsFuncs = append(srv.SizeInCellsAndPixelsFuncs, sizeInCellsAndPixelsFunc)
	}
	if positionerGet, ok := s.(interface {
		Cursor(tty TTY) (xPosCells, yPosCells uint, err error)
	}); ok {
		posGetFunc := func(tty TTY, _ Querier) (widthCells uint, heightCells uint, err error) {
			return positionerGet.Cursor(tty)
		}
		srv.posGetFuncs = append(srv.posGetFuncs, posGetFunc)
	}
	if positionerSet, ok := s.(interface {
		SetCursor(xPosCells, yPosCells uint, tty TTY) (err error)
	}); ok {
		posSetFunc := func(xPosCells, yPosCells uint, tty TTY, _ Querier) (err error) {
			return positionerSet.SetCursor(xPosCells, yPosCells, tty)
		}
		srv.posSetFuncs = append(srv.posSetFuncs, posSetFunc)
	}
	if !srv.avoidQuery {
		if cellSizer, ok := s.(interface {
			CellSizeQuery(qu Querier, tty TTY) (width, height float64, err error)
		}); ok {
			cellSizeFunc := func(tty TTY, qu Querier, _ Window) (width float64, height float64, err error) {
				return cellSizer.CellSizeQuery(qu, tty)
			}
			srv.cellSizeFuncs = append(srv.cellSizeFuncs, cellSizeFunc)
		}
		if sizerInCells, ok := s.(interface {
			SizeInCellsQuery(Querier, TTY) (widthCells, heightCells uint, err error)
		}); ok {
			sizeInCellsQueryFunc := func(tty TTY, qu Querier, _ Window) (widthCells, heightCells uint, err error) {
				return sizerInCells.SizeInCellsQuery(qu, tty)
			}
			srv.SizeInCellsFuncs = append(srv.SizeInCellsFuncs, sizeInCellsQueryFunc)
		}
		if sizerInPixels, ok := s.(interface {
			SizeInPixelsQuery(Querier, TTY) (widthPixels, heightPixels uint, err error)
		}); ok {
			sizerInPixelsQueryFunc := func(tty TTY, qu Querier, w Window) (widthPixels, heightPixels uint, err error) {
				return sizerInPixels.SizeInPixelsQuery(qu, tty)
			}
			srv.SizeInPixelsFuncs = append(srv.SizeInPixelsFuncs, sizerInPixelsQueryFunc)
		}
		if positionerGetQuery, ok := s.(interface {
			CursorQuery(qu Querier, tty TTY) (widthCells, heightCells uint, err error)
		}); ok {
			posGetFunc := func(tty TTY, qu Querier) (widthCells uint, heightCells uint, err error) {
				return positionerGetQuery.CursorQuery(qu, tty)
			}
			srv.posGetFuncs = append(srv.posGetFuncs, posGetFunc)
		}
		if positionerSetQuery, ok := s.(interface {
			SetCursorQuery(xPosCells, yPosCells uint, qu Querier, tty TTY) (err error)
		}); ok {
			posSetFunc := func(xPosCells, yPosCells uint, tty TTY, qu Querier) (err error) {
				return positionerSetQuery.SetCursorQuery(xPosCells, yPosCells, qu, tty)
			}
			srv.posSetFuncs = append(srv.posSetFuncs, posSetFunc)
		}
	}
	// add possibly inexact window checks at end
	if sizerInPixelsWindow, ok := s.(interface {
		SizeInPixelsWindow(w Window) (widthPixels, heightPixels uint, err error)
	}); ok {
		sizerInPixelsWindowFunc := func(_ TTY, _ Querier, w Window) (widthPixels uint, heightPixels uint, err error) {
			return sizerInPixelsWindow.SizeInPixelsWindow(w)
		}
		srv.SizeInPixelsFuncs = append(srv.SizeInPixelsFuncs, sizerInPixelsWindowFunc)
	}
	if wincher, ok := s.(interface {
		ResizeEvents(tty TTY) (_ <-chan Resolution, closeFunc func() error, _ error)
	}); ok {
		resizeEventFunc := func(tty TTY, _ Window) (_ <-chan Resolution, closeFunc func() error, _ error) {
			return wincher.ResizeEvents(tty)
		}
		srv.resizeEventFuncs = append(srv.resizeEventFuncs, resizeEventFunc)
	}
	if wincherWindow, ok := s.(interface {
		ResizeEventsWindow(w Window) (_ <-chan Resolution, closeFunc func() error, _ error)
	}); ok {
		resizeEventWindowFunc := func(_ TTY, w Window) (_ <-chan Resolution, closeFunc func() error, _ error) {
			return wincherWindow.ResizeEventsWindow(w)
		}
		srv.resizeEventFuncs = append(srv.resizeEventFuncs, resizeEventWindowFunc)
	}

	return srv
}

func (s *surveyor) CellSize(tty TTY, qu Querier, w Window, pr Properties) (width, height float64, err error) {
	if err := errors.NilReceiver(s); err != nil {
		return 0, 0, err
	}
	var errs []error
	for _, cellSizeFunc := range s.cellSizeFuncs {
		if cellSizeFunc == nil {
			continue
		}
		w, h, err := cellSizeFunc(tty, qu, w)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return w, h, nil
	}
	var wc, hc, wp, hp uint
	var hasSizeInCells, hasSizeInPixels bool
	for _, SizeInCellsAndPixelsFunc := range s.SizeInCellsAndPixelsFuncs {
		if SizeInCellsAndPixelsFunc == nil {
			continue
		}
		wc2, hc2, wp2, hp2, err := SizeInCellsAndPixelsFunc(tty, qu, w)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		hasSizeInCells2 := wc2 > 0 && hc2 > 0
		hasSizeInPixels2 := wp2 > 0 && hp2 > 0
		if hasSizeInCells2 {
			wc = wc2
			hc = hc2
			hasSizeInCells = true
		}
		if hasSizeInPixels2 {
			wp = wp2
			hp = hp2
			hasSizeInPixels = true
		}
		if hasSizeInCells && hasSizeInPixels {
			cellWidth := float64(wp) / float64(wc)
			cellHeight := float64(hp) / float64(hc)
			return cellWidth, cellHeight, nil
		}
		if !hasSizeInCells2 {
			errs = append(errs, errors.New(`received 0 length terminal sizes (in cells)`))
			continue
		}
		if !hasSizeInPixels2 {
			errs = append(errs, errors.New(`received 0 length terminal sizes (in pixels)`))
			continue
		}
	}
	if (hasSizeInCells || len(s.SizeInCellsFuncs) > 0) && (hasSizeInPixels || len(s.SizeInPixelsFuncs) > 0) {
		if !hasSizeInCells {
			for _, SizeInCellsFunc := range s.SizeInCellsFuncs {
				if SizeInCellsFunc == nil {
					continue
				}
				w, h, err := SizeInCellsFunc(tty, qu, w)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				if w > 0 && h > 0 {
					wc = w
					hc = h
					hasSizeInCells = true
					break
				}
				errs = append(errs, errors.New(`received 0 length terminal sizes (in cells)`))
			}
		}
		if !hasSizeInCells {
			errs = append(errs, errors.New("unable to query terminal resolution in cells"))
		} else if !hasSizeInPixels {
			for _, SizeInPixelsFunc := range s.SizeInPixelsFuncs {
				if SizeInPixelsFunc == nil {
					continue
				}
				w, h, err := SizeInPixelsFunc(tty, qu, w)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				if w > 0 && h > 0 {
					wp = w
					hp = h
					hasSizeInPixels = true
					break
				}
			}
		}
	}
	if hasSizeInCells && hasSizeInPixels {
		cellWidth := float64(wp) / float64(wc)
		cellHeight := float64(hp) / float64(hc)
		return cellWidth, cellHeight, nil
	}
	errRet := errors.Join(errs...)
	if errRet == nil {
		fmt.Printf("%+#v\n", s)
		errRet = errors.New("Surveyor.CellSize failed")
	} else {
		errRet = errors.Errorf("%s: %w", "Surveyor.CellSize failed", errRet)
	}
	return 0, 0, errRet
}
func (s *surveyor) SizeInCells(tty TTY, qu Querier, w Window, pr Properties) (width, height uint, err error) {
	if s == nil {
		return 0, 0, errors.NilReceiver()
	}
	var errs []error
	for _, SizeInCellsFunc := range s.SizeInCellsFuncs {
		if SizeInCellsFunc == nil {
			continue
		}
		w, h, err := SizeInCellsFunc(tty, qu, w)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return w, h, nil
	}
	for _, SizeInCellsAndPixelsFunc := range s.SizeInCellsAndPixelsFuncs {
		if SizeInCellsAndPixelsFunc == nil {
			continue
		}
		cw, ch, _, _, err := SizeInCellsAndPixelsFunc(tty, qu, w)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return cw, ch, nil
	}
	if len(s.cellSizeFuncs) > 0 && (len(s.SizeInPixelsFuncs) > 0 || len(s.SizeInCellsAndPixelsFuncs) > 0) {
		var cellWidth, cellHeight float64
		var widthInPixels, heightInPixels uint
		for _, cellSizeFunc := range s.cellSizeFuncs {
			if cellSizeFunc == nil {
				continue
			}
			w, h, err := cellSizeFunc(tty, qu, w)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			cellWidth = w
			cellHeight = h
			break
		}
		if cellWidth > 0 && cellHeight > 0 {
			for _, SizeInPixelsFunc := range s.SizeInPixelsFuncs {
				if SizeInPixelsFunc == nil {
					continue
				}
				wp, hp, err := SizeInPixelsFunc(tty, qu, w)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				widthInPixels = wp
				heightInPixels = hp
				goto divide
			}
			for _, SizeInCellsAndPixelsFunc := range s.SizeInCellsAndPixelsFuncs {
				if SizeInCellsAndPixelsFunc == nil {
					continue
				}
				_, _, wp, hp, err := SizeInCellsAndPixelsFunc(tty, qu, w)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				widthInPixels = wp
				heightInPixels = hp
				break
			}
		divide:
			widthInCells := uint(float64(widthInPixels) / cellWidth)
			heightInCells := uint(float64(heightInPixels) / cellHeight)
			return widthInCells, heightInCells, nil
		}
	}
	errRet := errors.Join(errs...)
	if errRet == nil {
		errRet = errors.New("Surveyor.SizeInCells failed")

	} else {
		errRet = errors.Errorf("%s: %w", "Surveyor.SizeInCells failed", errRet)
	}
	return 0, 0, errRet
}
func (s *surveyor) SizeInPixels(tty TTY, qu Querier, w Window, pr Properties) (width, height uint, err error) {
	if s == nil {
		return 0, 0, errors.NilReceiver()
	}
	var errs []error
	for _, SizeInPixelsFunc := range s.SizeInPixelsFuncs {
		if SizeInPixelsFunc == nil {
			continue
		}
		w, h, err := SizeInPixelsFunc(tty, qu, w)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return w, h, nil
	}
	var widthInCells, heightInCells uint
	for _, SizeInCellsAndPixelsFunc := range s.SizeInCellsAndPixelsFuncs {
		if SizeInCellsAndPixelsFunc == nil {
			continue
		}
		cw, ch, pw, ph, err := SizeInCellsAndPixelsFunc(tty, qu, w)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if cw > 0 && ch > 0 {
			widthInCells, heightInCells = cw, ch
		}
		if pw < 1 || ph < 1 {
			errs = append(errs, errors.New(`no pixel size`))
			continue
		}
		return pw, ph, nil
	}
	if len(s.cellSizeFuncs) > 0 &&
		((widthInCells > 0 && heightInCells > 0) || len(s.SizeInCellsFuncs) > 0 || len(s.SizeInCellsAndPixelsFuncs) > 0) {
		var cellWidth, cellHeight float64
		var widthInCells, heightInCells uint
		for _, cellSizeFunc := range s.cellSizeFuncs {
			if cellSizeFunc == nil {
				continue
			}
			w, h, err := cellSizeFunc(tty, qu, w)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			cellWidth = w
			cellHeight = h
			break
		}
		if cellWidth > 0 && cellHeight > 0 {
			if widthInCells > 0 && heightInCells > 0 {
				goto divide
			}
			for _, SizeInCellsFunc := range s.SizeInCellsFuncs {
				if SizeInCellsFunc == nil {
					continue
				}
				wc, hc, err := SizeInCellsFunc(tty, qu, w)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				widthInCells = wc
				heightInCells = hc
				goto divide
			}
			for _, SizeInCellsAndPixelsFunc := range s.SizeInCellsAndPixelsFuncs {
				if SizeInCellsAndPixelsFunc == nil {
					continue
				}
				wc, hc, _, _, err := SizeInCellsAndPixelsFunc(tty, qu, w)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				widthInCells = wc
				heightInCells = hc
				break
			}
		divide:
			widthInPixels := uint(float64(widthInCells) * cellWidth)
			heightInPixels := uint(float64(heightInCells) * cellHeight)
			return widthInPixels, heightInPixels, nil
		}
	}
	errRet := errors.Join(errs...)
	if errRet == nil {
		errRet = errors.New("Surveyor.SizeInPixels failed")

	} else {
		errRet = errors.Errorf("%s: %w", "Surveyor.SizeInPixels failed", errRet)
	}
	return 0, 0, errRet
}
func (s *surveyor) Cursor(tty TTY, qu Querier, _ Window, _ Properties) (xPosCells, yPosCells uint, err error) {
	if s == nil {
		return 0, 0, errors.NilReceiver()
	}
	var errs []error
	for _, posGetFunc := range s.posGetFuncs {
		if posGetFunc == nil {
			continue
		}
		widthPixels, heightPixels, err := posGetFunc(tty, qu)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return widthPixels, heightPixels, nil

	}
	errRet := errors.Join(errs...)
	if errRet == nil {
		errRet = errors.New("Surveyor.Cursor failed")

	} else {
		errRet = errors.Errorf("%s: %w", "Surveyor.Cursor failed", errRet)
	}
	return 0, 0, errRet
}
func (s *surveyor) SetCursor(xPosCells, yPosCells uint, tty TTY, qu Querier, w Window, pr Properties) (err error) {
	if s == nil {
		return errors.NilReceiver()
	}
	var errs []error
	for _, posSetFunc := range s.posSetFuncs {
		if posSetFunc == nil {
			continue
		}
		err := posSetFunc(xPosCells, yPosCells, tty, qu)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return nil

	}
	errRet := errors.Join(errs...)
	if errRet == nil {
		errRet = errors.New("Surveyor.SetCursor failed")

	} else {
		errRet = errors.Errorf("%s: %w", "Surveyor.SetCursor failed", errRet)
	}
	return errRet
}
func (s *surveyor) WatchResizeEventsStart(tty TTY, qu Querier, w Window, pr Properties) (_ <-chan Resolution, closeFunc func() error, _ error) {
	var resChans []<-chan Resolution
	var closeFuncs []func() error
	var errs []error
	if wincher, ok := tty.(interface {
		ResizeEvents() (_ <-chan Resolution, closeFunc func() error, _ error)
	}); ok {
		winch, closeFunc, err := wincher.ResizeEvents()
		if err == nil {
			if winch != nil {
				resChans = append(resChans, winch)
			}
			if closeFunc != nil {
				closeFuncs = append(closeFuncs, closeFunc)
			}
		} else {
			errs = append(errs, err)
		}
	}
	for _, winchFunc := range s.resizeEventFuncs {
		winch, closeFunc, err := winchFunc(tty, w)
		if err == nil {
			if winch != nil {
				resChans = append(resChans, winch)
			}
			if closeFunc != nil {
				closeFuncs = append(closeFuncs, closeFunc)
			}
		} else {
			errs = append(errs, err)
		}
	}
	var comb chan Resolution
	if len(resChans) > 0 {
		comb = make(chan Resolution)
		for _, winch := range resChans {
			go func(winch <-chan Resolution) {
				for {
					if winch == nil {
						break
					}
					res, ok := <-winch
					if !ok {
						break
					}
					comb <- res
				}
			}(winch)
		}
	}
	closeOnce := sync.Once{}
	closeFunc = func() error {
		var errRet error
		closeOnce.Do(func() {
			var errs []error
			for _, closeFunc := range closeFuncs {
				errs = append(errs, closeFunc())
			}
			close(comb)
			errRet = errors.Join(errs...)
		})
		return errRet
	}
	s.watchWINCHStopFunc = closeFunc
	errRet := errors.Join(errs...)
	return comb, closeFunc, errRet
}
func (s *surveyor) WatchResizeEventsStop() error {
	if s == nil || s.watchWINCHStopFunc == nil {
		return nil
	}
	return s.watchWINCHStopFunc()
}
