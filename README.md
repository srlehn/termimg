[![PkgGoDev](https://pkg.go.dev/badge/github.com/srlehn/termimg)](https://pkg.go.dev/github.com/srlehn/termimg@master)
[![Go Report Card](https://goreportcard.com/badge/srlehn/termimg)](https://goreportcard.com/report/srlehn/termimg)
![Lines of code](https://img.shields.io/tokei/lines/github/srlehn/termimg)
[![MIT license](https://img.shields.io/badge/License-MIT-blue.svg)](https://lbesson.mit-license.org/)

# TermImg

termimg tries to draw images into terminals.

**VERY EXPERIMENTAL!!**

implemented drawing methods: sixel, iTerm2, kitty, Terminology, DomTerm, urxvt, X11, GDI+, block characters

<details open><summary><h2>example cli tool</h2></summary>

<blockquote><details open><summary><h3>timg cli tool</h3></summary>

```sh
go install github.com/srlehn/termimg/cmd/timg@latest
timg --debug=true runterm -t mlterm -d sixel -p 10,10,15x15 picture.png
```
</details>

<blockquote></details>


<details open><summary><h2>library usage</h2></summary>

<blockquote><details><summary><h3>one-off image draw</h3></summary>

```go
import (
    _ "github.com/srlehn/termimg/drawers"
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
    _ "github.com/srlehn/termimg/drawers"
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

<details><summary><h3>advanced</h3></summary>

```go
import (
    _ "github.com/srlehn/termimg/drawers"
	_ "github.com/srlehn/termimg/terminals"
)

func main(){
	wm.SetImpl(wmimpl.Impl())
	cr := &term.Creator{
		PTYName:         `dev/pts/2`,
		TTYProvFallback: gotty.New,
		Querier:         qdefault.NewQuerier(),
		Resizer:         &rdefault.Resizer{},
	}
	tm, err := term.NewTerminal(cr)
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
