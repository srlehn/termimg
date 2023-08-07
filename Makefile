SRC=$(shell find . -type d \( -path ./vendor -o -path ./testdata \) -prune -o -name '*.go' -print)

ifeq ($(GOOS),windows)
EXT = ".exe"
endif


.PHONY: build
build: timg

.PHONY: all
all: timg termui


timg: ${SRC}
	@env GOWORK=off CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags '-w -s' -o timg${EXT} ./cmd/timg

.PHONY: timg-caire
timg-caire: ${SRC}
	@env GOWORK=off CGO_ENABLED=1 go build -tags 'caire' -trimpath -ldflags '-w -s' -o timg${EXT} ./cmd/timg

# termui: *.go cmd/termui_test/main.go
termui: ${SRC}
	@env GOWORK=off CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags '-w -s' -o termui${EXT} cmd/termui_test/main.go


.PHONY: dev
dev: ${SRC}
	@CGO_ENABLED=0 go build -tags 'dev' -trimpath -ldflags '-w -s' -o timg${EXT} ./cmd/timg

.PHONY: dev-caire
dev-caire: ${SRC}
	@env GOWORK=off CGO_ENABLED=1 go build -tags 'dev,caire' -trimpath -ldflags '-w -s' -o timg${EXT} ./cmd/timg




.PHONY: clean
clean:
	@rm -f -- timg timg.exe termui termui.exe


# line count

.PHONY: cloc-wc
cloc-wc:
	@wc -l ${SRC} | tail -n 1 | sed 's,^ *,,;s, .*,,'

.PHONY: cloc-cloc
cloc-cloc:
	@cloc --include-lang=Go --quiet --hide-rate --fullpath --not-match-d='(vendor|testdata)' --json . | jq -Cr '.Go.code'

.PHONY: cloc-gocloc
cloc-gocloc:
	@gocloc --include-lang=Go --not-match-d='(vendor|testdata)' --output-type=json . | jq -Cr '.languages[] | select(.name == "Go") | .code'

.PHONY: cloc-scc
cloc-scc:
	@scc -M '(^vendor/|^testdata/|_test.go$$)' --binary --no-gen -f json | jq -Cr '.[] | select(.Name == "Go") | .Code'

.PHONY: cloc-tokei
cloc-tokei:
	@tokei -e vendor -e testdata -t Go -o json | jq -Cr '.Go.code'


.PHONY: install-gocloc
install-gocloc:
	@go install github.com/hhatto/gocloc/cmd/gocloc@latest

.PHONY: install-scc
install-scc:
	@go install github.com/boyter/scc/v3@latest

.PHONY: install-tokei
install-tokei:
	@cargo install tokei
