//go:build dev

package tcellimg

import "github.com/gdamore/tcell/v2/views"

var _ views.Widget = (*Image)(nil)

type Image struct {
	*views.CellView
}

func ff() {
	views.CellView
}
