# Go Pty

<p>
    <a href="https://github.com/aymanbagabas/go-pty/releases"><img src="https://img.shields.io/github/release/aymanbagabas/go-pty.svg" alt="Latest Release"></a>
    <a href="https://pkg.go.dev/github.com/aymanbagabas/go-pty?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="GoDoc"></a>
    <a href="https://github.com/aymanbagabas/go-pty/actions"><img src="https://github.com/aymanbagabas/go-pty/workflows/build/badge.svg" alt="Build Status"></a>
</p>

Go-Pty is a package for using pseudo-terminal interfaces in Go. It supports Unix PTYs and Windows through [ConPty](https://learn.microsoft.com/en-us/windows/console/creating-a-pseudoconsole-session).

## Why can't we just use os/exec?

Windows requires updating the process running in the PTY with a [special attribute](https://learn.microsoft.com/en-us/windows/console/creating-a-pseudoconsole-session) to enable ConPty support. This is not possible with os/exec see [go#62708](https://github.com/golang/go/issues/62708) and [go#6271](https://github.com/golang/go/pull/62710). On Unix, `pty.Cmd` is just a wrapper around `os/exec.Cmd` that sets up the PTY.

## Usage

```sh
go get github.com/aymanbagabas/go-pty
```

Example running `grep`

```go
package main

import (
	"io"
	"log"
	"os"

	"github.com/aymanbagabas/go-pty"
)

func main() {
	pty, err := pty.New()
	if err != nil {
		log.Fatalf("failed to open pty: %s", err)
	}

	defer pty.Close()
	c := pty.Command("grep", "--color=auto", "bar")
	if err := c.Start(); err != nil {
		log.Fatalf("failed to start: %s", err)
	}

	go func() {
		pty.Write([]byte("foo\n"))
		pty.Write([]byte("bar\n"))
		pty.Write([]byte("baz\n"))
		pty.Write([]byte{4}) // EOT
	}()
	go io.Copy(os.Stdout, pty)

	if err := c.Wait(); err != nil {
		panic(err)
	}
}
```

Refer to [./examples](./examples) for more examples.

## Credits

- [creack/pty](https://github.com/creack/pty/): support for Unix PTYs
- [microsoft/hcsshim](https://github.com/microsoft/hcsshim): Windows ConPty implementation

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) for details.
