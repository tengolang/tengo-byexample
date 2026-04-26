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
	arg := args[0]
	var output, errMsg string
	if arg.Type() == js.TypeObject {
		files := make(map[string]string)
		keys := js.Global().Get("Object").Call("keys", arg)
		for i := 0; i < keys.Length(); i++ {
			k := keys.Index(i).String()
			files[k] = arg.Get(k).String()
		}
		output, errMsg = tengorunner.RunFiles(files)
	} else {
		output, errMsg = tengorunner.Run(arg.String())
	}
	return map[string]any{"output": output, "error": errMsg}
}
