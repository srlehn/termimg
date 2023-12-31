# go-sixel

DRCS Sixel Encoder/Decoder

![](http://go-gyazo.appspot.com/75ec3ce96dfc573e.png)

## Installation

```
$ go get github.com/mattn/go-sixel
```

You can install gosr (go sixel renderer), gosd (go sixel decoder) with following installation instruction.

```
$ go get github.com/mattn/go-sixel/cmd/gosr
$ go get github.com/mattn/go-sixel/cmd/gosd
```

|Command|Description         |
|-------|--------------------|
|gosr   |Image renderer      |
|gosd   |Decoder to png      |
|goscat |Render cats         |
|gosgif |Render animation GIF|
|gosl   |Run SL              |

## Usage

Encode
```
$ cat foo.png | gosr -
```

Decode

```
$ cat foo.drcs | gosd > foo.png
```

Use as library

```go
img, _, _ := image.Decode(filename)
sixel.NewEncoder(os.Stdout).Encode(img)
```

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a mattn)
