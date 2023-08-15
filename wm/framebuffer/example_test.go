// Copyright 2013 Konstantin Kulikov. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package framebuffer_test

import (
	"fmt"
	"image/color"
	"log"

	"github.com/srlehn/termimg/wm/framebuffer"
)

func Example() {
	fb, err := framebuffer.Init("/dev/fb0")
	if err != nil {
		log.Fatalln(err)
	}
	defer fb.Close()
	fb.Clear(color.RGBA{})
	fb.Set(200, 100, color.RGBA{R: 255})
	fmt.Scanln()
}
