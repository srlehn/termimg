// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

package framebuffer

// <linux/vt.h>

type vt_mode struct {
	mode   int8  // vt mode
	waitv  int8  // If set, hang on writes if not active.
	relsig int16 // signal to raise on release req
	acqsig int16 // signal to raise on acquisition
	frsig  int16 // unused (set to 0)
}

type vt_stat struct {
	v_active uint16 // active vt
	v_signal uint16 // signal to send
	v_state  uint16 // vt bitmask
}

const (
	_VT_GETMODE    = 0x5601 // get mode of active vt
	_VT_SETMODE    = 0x5602 // set mode of active vt
	_VT_GETSTATE   = 0x5603 // get global vt state info
	_VT_RELDISP    = 0x5605 // release display
	_VT_ACTIVATE   = 0x5606 // make vt active
	_VT_WAITACTIVE = 0x5607 // wait for vt active
	_VT_PROCESS    = 0x01   // process controls switching
	_VT_ACKACQ     = 0x02   // acknowledge switch
)
