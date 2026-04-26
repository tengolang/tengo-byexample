// Package tengorunner wraps the Tengo script engine so that the tengo
// dependency is visible to go mod tidy on all platforms, not only when
// building for js/wasm.
package tengorunner

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tengolang/tengo/v3"
	"github.com/tengolang/tengo/v3/stdlib"
)

// Run compiles and executes a single script, capturing all fmt output.
func Run(code string) (output string, errMsg string) {
	return RunFiles(map[string]string{"main": code})
}

// RunFiles compiles and executes a set of files. The file named "main" is the
// entry point; all others are registered as importable source modules so that
// import("math_utils") resolves against the provided map.
func RunFiles(files map[string]string) (output string, errMsg string) {
	main, ok := files["main"]
	if !ok {
		return "", "no main file provided"
	}
	var buf bytes.Buffer
	mods := CaptureFmtModuleMap(&buf)
	for name, src := range files {
		if name != "main" {
			mods.AddSourceModule(name, []byte(src))
		}
	}
	s := tengo.NewScript([]byte(main))
	s.SetImports(mods)
	s.SetMaxAllocs(1 << 20)
	compiled, err := s.Compile()
	if err != nil {
		return "", err.Error()
	}
	if err := compiled.Run(); err != nil {
		return buf.String(), err.Error()
	}
	return buf.String(), ""
}

// CaptureFmtModuleMap returns a module map identical to the standard one
// except that the fmt module writes to w instead of os.Stdout, and the os
// module is backed by an in-memory filesystem for WASM sandbox safety.
func CaptureFmtModuleMap(w *bytes.Buffer) *tengo.ModuleMap {
	mods := stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	mods.AddBuiltinModule("os", mockOsModule())
	mods.AddBuiltinModule("fmt", map[string]tengo.Object{
		"print": &tengo.UserFunction{Name: "print", Value: func(args ...tengo.Object) (tengo.Object, error) {
			pa, err := collectPrintArgs(args)
			if err != nil {
				return nil, err
			}
			_, _ = fmt.Fprint(w, strings.Join(pa, ""))
			return nil, nil
		}},
		"println": &tengo.UserFunction{Name: "println", Value: func(args ...tengo.Object) (tengo.Object, error) {
			pa, err := collectPrintArgs(args)
			if err != nil {
				return nil, err
			}
			_, _ = fmt.Fprintln(w, strings.Join(pa, ""))
			return nil, nil
		}},
		"printf": &tengo.UserFunction{Name: "printf", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, tengo.ErrWrongNumArguments
			}
			format, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "format", Expected: "string", Found: args[0].TypeName()}
			}
			if len(args) == 1 {
				_, _ = fmt.Fprint(w, format.Value)
				return nil, nil
			}
			s, err := tengo.Format(format.Value, args[1:]...)
			if err != nil {
				return nil, err
			}
			_, _ = fmt.Fprint(w, s)
			return nil, nil
		}},
		"sprintf": &tengo.UserFunction{Name: "sprintf", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, tengo.ErrWrongNumArguments
			}
			format, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "format", Expected: "string", Found: args[0].TypeName()}
			}
			if len(args) == 1 {
				return format, nil
			}
			s, err := tengo.Format(format.Value, args[1:]...)
			if err != nil {
				return nil, err
			}
			return &tengo.String{Value: s}, nil
		}},
	})
	return mods
}

func collectPrintArgs(args []tengo.Object) ([]string, error) {
	result := make([]string, 0, len(args))
	total := 0
	for _, arg := range args {
		s, _ := tengo.ToString(arg)
		if total+len(s) > tengo.MaxStringLen {
			return nil, tengo.ErrStringLimit
		}
		total += len(s)
		result = append(result, s)
	}
	return result, nil
}
