[![PkgGoDev](https://pkg.go.dev/badge/github.com/srlehn/termimg)](https://pkg.go.dev/github.com/srlehn/termimg@master)
[![Go Report Card](https://goreportcard.com/badge/srlehn/termimg)](https://goreportcard.com/report/srlehn/termimg)
![Lines of code](https://img.shields.io/tokei/lines/github/srlehn/termimg)
[![MIT license](https://img.shields.io/badge/License-MIT-blue.svg)](https://lbesson.mit-license.org/)

# TermImg

termimg tries to draw images into terminals.

The rectangular drawing area is given in cell coordinates (not pixels). Origin is the upper left corner.

**VERY EXPERIMENTAL!!**

implemented drawing methods: sixel, iTerm2, kitty, Terminology, DomTerm, urxvt, X11, GDI+, block characters

<details open><summary><h2>Example CLI Tool</h2></summary>

<blockquote><details open><summary><h3><span style="color:mediumseagreen">timg</span> CLI Tool</h3></summary>

installation:
```sh
go install github.com/srlehn/termimg/cmd/timg@master
```
The cell coordinates are passed in this format: `<x>,<y>,<w>x<h>` where x is the column, y the row, w the width and h the height.

The `show` command draws the image in the current terminal:
```sh
timg show -p 10,10,15x15 picture.png
```
If an error occurs the `--debug=true` argument shows where in the code it happens.

The `runterm` command starts the terminal specified with the `-t` flag. If no drawer is enforced by the optional `-d` flag, the best fitting one is used. This command is probably only useful for testing.
```sh
timg --debug=true runterm -t mlterm -d sixel -p 10,10,15x15 picture.png
```
</details>

<blockquote></details>

<details open><summary><h2>Library Usage</h2></summary>

<blockquote><details><summary><h3>One-Off Image Draw</h3></summary>

```go
import (
    _ "github.com/srlehn/termimg/drawers/all"
	_ "github.com/srlehn/termimg/terminals"
)

func main(){
    defer termimg.CleanUp()
    _ = termimg.DrawFile(`picture.png`, image.Rect(10,10,40,25))
}
```
</details>

---

<details open><summary><h3>with NewImage…()</h3></summary>

For repeated image drawing create a term.Image via the NewImage…() functions.
This allows caching of the encoded image.

```go
import (
    _ "github.com/srlehn/termimg/drawers/all"
	_ "github.com/srlehn/termimg/terminals"
)

func main(){
	tm, _ := termimg.Terminal()
    defer tm.Close()
    timg := termimg.NewImageFileName(`picture.png`)
    _ = tm.Draw(timg, image.Rect(10,10,40,25))
}
```
</details>

---

<details><summary><h3>Advanced</h3></summary>

```go
import (
    _ "github.com/srlehn/termimg/drawers/sane"
	_ "github.com/srlehn/termimg/terminals"
)

func main(){
	wm.SetImpl(wmimpl.Impl())
	opts := &term.Options{
		PTYName:         `dev/pts/2`,
		TTYProvFallback: gotty.New,
		Querier:         qdefault.NewQuerier(),
		Resizer:         &rdefault.Resizer{},
	}
	tm, err := term.NewTerminal(opts)
	if err != nil {
		log.Fatal(err)
	}
    defer tm.Close()
    var img image.Image // TODO load image
    timg := termimg.NewImage(img)
    if err := tm.Draw(timg, image.Rect(10,10,40,25)); err != nil {
		log.Fatal(err)
    }
}
```
</details>

</blockquote></details>
