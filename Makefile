GOROOT        := $(shell go env GOROOT)
# wasm_exec.js moved from misc/wasm/ to lib/wasm/ in Go 1.24.
WASM_EXEC     := $(firstword $(foreach f,\
	$(GOROOT)/lib/wasm/wasm_exec.js \
	$(GOROOT)/misc/wasm/wasm_exec.js,$(wildcard $(f))))

.PHONY: all build clean serve

all: build

build: docs/tengo.wasm.gz docs/wasm_exec.js docs/site.css docs/site.js
	go run ./cmd/build/

docs/tengo.wasm.gz: cmd/wasm/main.go go.mod go.sum
	mkdir -p docs
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o docs/tengo.wasm ./cmd/wasm/
	gzip -9 -f docs/tengo.wasm

docs/wasm_exec.js: $(WASM_EXEC)
	mkdir -p docs
	cp $(WASM_EXEC) docs/wasm_exec.js

docs/site.css: static/site.css
	mkdir -p docs
	cp static/site.css docs/site.css

docs/site.js: static/site.js
	mkdir -p docs
	cp static/site.js docs/site.js

clean:
	rm -rf docs/

serve:
	python3 -m http.server -d docs 8080
