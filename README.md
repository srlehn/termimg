# TermImg

termimg tries to draw images into terminals.

**VERY EXPERIMENTAL!!**

## example cli tool

```sh
go install github.com/srlehn/termimg/cmd/timg@latest
timg --debug=true runterm -t mlterm -d sixel -p 10,10,15x15 picture.png
```

## library usage

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

### advanced

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
