# TermImg

<!-- this readme is meant to be displayed in an
HTML-capable markdown pager (github via browser, etc) -->

**termimg** draws images into terminals.
This repository includes the **termimg** library and
the **timg** cli tool which can be used in shell scripts.

[![PkgGoDev](https://pkg.go.dev/badge/github.com/srlehn/termimg)](https://pkg.go.dev/github.com/srlehn/termimg@master)
[![Go Report Card](https://goreportcard.com/badge/srlehn/termimg)](https://goreportcard.com/report/srlehn/termimg)
![Lines of code](https://tokei.rs/b1/github/srlehn/termimg?type=Go&category=code)
[![MIT license](https://img.shields.io/badge/License-MIT-blue.svg)](https://lbesson.mit-license.org/)
![experimental](https://img.shields.io/badge/status-experimental-orange.svg)

This module is still **experimental**.
Most parts need more testing on platforms other than my own (Debian stable, X11).
The API of some unfinished packages will change.
There are still many small issues to be fixed. Please report them.

Images are being placed by using **cell** not pixel **coordinates**.
(The origin is the upper left terminal corner just like with image.Image.)
The latter doesn't make sense in the context of a terminal.
termimg is able to fit images into cell boundaries
while preserving their aspect ratio.

termimg implements **several drawing methods**:
<ins>
[sixel](https://en.wikipedia.org/wiki/Sixel),
[iTerm2](https://iterm2.com/documentation-images.html),
[kitty](https://sw.kovidgoyal.net/kitty/graphics-protocol/),
[Terminology](https://git.enlightenment.org/enlightenment/terminology#extended-escapes-for-terminology-only),
[DomTerm](https://domterm.org/Wire-byte-protocol.html#Miscellaneous-sequences),
[urxvt](https://manpages.ubuntu.com/manpages/jammy/man1/urxvt-background.1.html#old%20background%20image%20settings),
X11,
GDI+,
block characters
</ins> - by default the most appropriate one will be chosen.

termimg is highly modular, most parts are exchangeable.
Multiple term.TTY implementations exist for tty libraries used in common TUI frameworks.

The root package **"termimg"** contains an easy to use high-level API
with a lot of default dependencies.
Direct usage of the core package **"/term"** is recommended
for more control and nearly no external dependencies.
The **"/drawers/sane"** drawer collection will only use actual drawing methods,
**"/drawers/all"** will try to produce an image by any means.

<details open><summary><h2>timg - tool for the CLI</h2></summary>

<blockquote><details open>
<summary>demo gifs</summary>

![demo_pic_ls.gif](https://raw.githubusercontent.com/srlehn/termimg/master/_demos/demo_pic_ls.gif)
![demo_vid.gif](https://raw.githubusercontent.com/srlehn/termimg/master/_demos/demo_vid.gif)
</details>

installation:

```sh
go install github.com/srlehn/termimg/cmd/timg@master
```

usage:

```text
$ timg
timg display terminal graphics

Usage:
  timg [flags]
  timg [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  list        list images
  properties  list terminal properties
  query       query terminal
  resolution  print terminal resolution
  scale       fit pixel area into a cell area while maintaining scale
  show        display image

Flags:
  -d, --debug             debug errors
  -h, --help              help for timg
  -l, --log-file string   log file
  -s, --silent            silence errors

Use "timg [command] --help" for more information about a command.
```

The `list` command displays thumbnails of previewable files
for a given directory similar to "lsix":

```sh
timg list ~/Pictures
```

The `show` command draws the image in the current terminal:

```sh
timg show -p 10,10,15x15 picture.png
```

Cell coordinates are optional for the show command,
they are passed in this format: `<x>,<y>,<w>x<h>`
where x is the column, y the row, w the width and h the height.

If an error occurs the `--debug/-d` flag shows where in the code it happens.

The `runterm` command starts the terminal specified with the `-t` flag.
If no drawer is enforced by the optional `-d` flag, the best fitting one is used.
This command is meant for testing purposes.

```sh
timg -d runterm -t mlterm -d sixel -p 10,10,15x15 picture.png
```

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
    opts := []term.Option{
        term.SetLogFile(`termimg.log`, true),
        term.SetPTYName(`dev/pts/2`),
        term.SetTTYProvider(gotty.New, false),
        term.SetQuerier(qdefault.NewQuerier(), true),
        term.SetWindowProvider(wm.SetImpl(wmImplementation), true),
        term.SetResizer(&rdefault.Resizer{}),
    }
    tm, err := term.NewTerminal(opts...)
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

The default options are packed together in `termimg.DefaultConfig`.
</details>

</blockquote></details>
