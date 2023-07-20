SRC=$(shell find . -type d \( -path ./vendor -o -path ./testdata \) -prune -o -name '*.go' -print)

ifeq ($(GOOS),windows)
EXT = ".exe"
endif


.PHONY: build
build: timg

.PHONY: all
all: timg termui


timg: ${SRC}
	@CGO_ENABLED=0 go build -trimpath -ldflags '-w -s' -o timg${EXT} ./cmd/timg

# termui: *.go cmd/termui_test/main.go
termui: ${SRC}
	@CGO_ENABLED=0 go build -trimpath -ldflags '-w -s' -o termui${EXT} cmd/termui_test/main.go


.PHONY: dev
dev: ${SRC}
	@CGO_ENABLED=0 go build -tags 'dev' -trimpath -ldflags '-w -s' -o timg${EXT} ./cmd/timg


.PHONY: clean
clean:
	@rm -f -- timg termui

.PHONY: cloc
cloc:
	@wc -l ${SRC} | tail -n 1 | sed 's,^ *,,;s, .*, lines,'
