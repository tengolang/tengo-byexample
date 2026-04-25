//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/tengolang/tengo-byexample/internal/tengorunner"
)

func main() {
	js.Global().Set("tengoRun", js.FuncOf(tengoRun))
	<-make(chan struct{})
}

func tengoRun(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return map[string]any{"output": "", "error": "no code provided"}
	}
	output, errMsg := tengorunner.Run(args[0].String())
	return map[string]any{"output": output, "error": errMsg}
}
