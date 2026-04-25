package tengorunner

import (
	"os"

	"github.com/tengolang/tengo/v3"
)

// mockOsModule returns an "os" builtin module backed by an in-memory
// filesystem and a mutable environment map, so that OS-oriented examples
// work inside the browser WASM sandbox without real filesystem access.
func mockOsModule() map[string]tengo.Object {
	vfs := &virtualFS{files: make(map[string][]byte)}
	env := map[string]string{
		"HOME":  "/home/tengo",
		"USER":  "tengo",
		"PATH":  "/usr/bin:/bin",
		"SHELL": "/bin/sh",
	}

	return map[string]tengo.Object{
		// platform constants
		"platform": &tengo.String{Value: "js"},
		"arch":     &tengo.String{Value: "wasm"},

		// file-open flag constants (same values as real os package)
		"o_rdonly": tengo.Int{Value: int64(os.O_RDONLY)},
		"o_wronly": tengo.Int{Value: int64(os.O_WRONLY)},
		"o_rdwr":   tengo.Int{Value: int64(os.O_RDWR)},
		"o_append": tengo.Int{Value: int64(os.O_APPEND)},
		"o_create": tengo.Int{Value: int64(os.O_CREATE)},
		"o_excl":   tengo.Int{Value: int64(os.O_EXCL)},
		"o_trunc":  tengo.Int{Value: int64(os.O_TRUNC)},

		// args returns a fixed argv so examples that inspect args work.
		"args": &tengo.UserFunction{Name: "args", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return &tengo.Array{Value: []tengo.Object{
				&tengo.String{Value: "tengo"},
				&tengo.String{Value: "playground.tengo"},
			}}, nil
		}},

		// getenv / setenv / unsetenv / lookup_env
		"getenv": &tengo.UserFunction{Name: "getenv", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			return &tengo.String{Value: env[k.Value]}, nil
		}},
		"setenv": &tengo.UserFunction{Name: "setenv", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return nil, tengo.ErrWrongNumArguments
			}
			k, ok1 := args[0].(*tengo.String)
			v, ok2 := args[1].(*tengo.String)
			if !ok1 || !ok2 {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			env[k.Value] = v.Value
			return tengo.UndefinedValue, nil
		}},
		"unsetenv": &tengo.UserFunction{Name: "unsetenv", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			delete(env, k.Value)
			return tengo.UndefinedValue, nil
		}},
		"lookup_env": &tengo.UserFunction{Name: "lookup_env", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			k, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			val, found := env[k.Value]
			if found {
				return &tengo.Array{Value: []tengo.Object{&tengo.String{Value: val}, tengo.TrueValue}}, nil
			}
			return &tengo.Array{Value: []tengo.Object{&tengo.String{Value: ""}, tengo.FalseValue}}, nil
		}},

		// create opens a path for writing (truncating), returns a file object.
		"create": &tengo.UserFunction{Name: "create", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			p, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			return vfs.writeFile(p.Value, false), nil
		}},

		// open opens a path for reading.
		"open": &tengo.UserFunction{Name: "open", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			p, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			return vfs.readFile(p.Value), nil
		}},

		// open_file respects O_APPEND so writing-files.tengo works.
		"open_file": &tengo.UserFunction{Name: "open_file", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) < 2 {
				return nil, tengo.ErrWrongNumArguments
			}
			p, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			flags, ok2 := args[1].(tengo.Int)
			if !ok2 {
				return nil, tengo.ErrInvalidArgumentType{Name: "second", Expected: "int", Found: args[1].TypeName()}
			}
			append := flags.Value&int64(os.O_APPEND) != 0
			return vfs.writeFile(p.Value, append), nil
		}},

		// read_file reads the full contents of a virtual file.
		"read_file": &tengo.UserFunction{Name: "read_file", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			p, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			data, exists := vfs.files[p.Value]
			if !exists {
				return nil, tengo.ErrIndexOutOfBounds
			}
			cp := make([]byte, len(data))
			copy(cp, data)
			return &tengo.Bytes{Value: cp}, nil
		}},

		// remove deletes a virtual file.
		"remove": &tengo.UserFunction{Name: "remove", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			p, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			delete(vfs.files, p.Value)
			return tengo.UndefinedValue, nil
		}},

		// exit is a no-op in the browser playground.
		"exit": &tengo.UserFunction{Name: "exit", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return tengo.UndefinedValue, nil
		}},
	}
}

// virtualFS is an in-memory filesystem used by mockOsModule.
type virtualFS struct {
	files map[string][]byte
}

// writeFile returns a file-like Tengo map for writing. If appendMode is true
// and the file already exists, new data is appended; otherwise it is truncated.
func (vfs *virtualFS) writeFile(path string, appendMode bool) tengo.Object {
	if !appendMode {
		vfs.files[path] = []byte{}
	} else if _, ok := vfs.files[path]; !ok {
		vfs.files[path] = []byte{}
	}

	writeStr := func(s string) {
		vfs.files[path] = append(vfs.files[path], []byte(s)...)
	}

	return &tengo.Map{Value: map[string]tengo.Object{
		"write_string": &tengo.UserFunction{Name: "write_string", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "string", Found: args[0].TypeName()}
			}
			writeStr(s.Value)
			return tengo.Int{Value: int64(len(s.Value))}, nil
		}},
		"write": &tengo.UserFunction{Name: "write", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			b, ok := args[0].(*tengo.Bytes)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "bytes", Found: args[0].TypeName()}
			}
			vfs.files[path] = append(vfs.files[path], b.Value...)
			return tengo.Int{Value: int64(len(b.Value))}, nil
		}},
		"close": &tengo.UserFunction{Name: "close", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return tengo.UndefinedValue, nil
		}},
		"name": &tengo.UserFunction{Name: "name", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return &tengo.String{Value: path}, nil
		}},
	}}
}

// readFile returns a file-like Tengo map for reading with a position cursor.
func (vfs *virtualFS) readFile(path string) tengo.Object {
	data := vfs.files[path]
	pos := new(int)

	readInto := func(buf *tengo.Bytes) tengo.Object {
		n := copy(buf.Value, data[*pos:])
		*pos += n
		return tengo.Int{Value: int64(n)}
	}

	return &tengo.Map{Value: map[string]tengo.Object{
		"read": &tengo.UserFunction{Name: "read", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			buf, ok := args[0].(*tengo.Bytes)
			if !ok {
				return nil, tengo.ErrInvalidArgumentType{Name: "first", Expected: "bytes", Found: args[0].TypeName()}
			}
			return readInto(buf), nil
		}},
		"close": &tengo.UserFunction{Name: "close", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return tengo.UndefinedValue, nil
		}},
		"name": &tengo.UserFunction{Name: "name", Value: func(args ...tengo.Object) (tengo.Object, error) {
			return &tengo.String{Value: path}, nil
		}},
	}}
}

