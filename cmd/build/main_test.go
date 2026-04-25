package main

import (
	"testing"
)

func TestParseExample(t *testing.T) {
	src := `// # Hello World
//
// Our first Tengo program prints the classic "Hello, World!" greeting.
// Every example on this site can be edited and run live in the browser.

// We start by importing the fmt module.
fmt := import("fmt")

// Then we use it to print.
fmt.println("Hello, World!")
`
	got := parseExample("examples/01-basics/001-hello-world.tengo", []byte(src))

	wantTitle := "Hello World"
	wantDescription := "Our first Tengo program prints the classic \"Hello, World!\" greeting. Every example on this site can be edited and run live in the browser."

	if got.Title != wantTitle {
		t.Errorf("Title = %q, want %q", got.Title, wantTitle)
	}
	if got.Description != wantDescription {
		t.Errorf("Description = %q, want %q", got.Description, wantDescription)
	}

	for i, s := range got.Sections {
		t.Logf("Section %d: Doc=%q Code=%q", i, s.Doc, s.Code)
	}

	if len(got.Sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(got.Sections))
	}
}

func TestParseExampleIndented(t *testing.T) {
	src := `// # Indented
//
// Description.

func main() {
    // Indented comment
    fmt.println("hi")
}
`
	got := parseExample("test.tengo", []byte(src))

	// We expect 3 sections:
	// 1. func main() {
	// 2. Doc: Indented comment, Code: fmt.println("hi")
	// 3. Code: }
	// (or however the parser currently groups them)

	foundIndented := false
	for _, s := range got.Sections {
		if s.Doc == "Indented comment" {
			foundIndented = true
			break
		}
	}

	if !foundIndented {
		t.Errorf("did not find extracted indented comment")
		for i, s := range got.Sections {
			t.Logf("Section %d: Doc=%q Code=%q", i, s.Doc, s.Code)
		}
	}
}

func TestFileSlug(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"examples/01-basics/001-hello-world.tengo", "hello-world"},
		{"hello.tengo", "hello"},
		{"005-switch.tengo", "switch"},
	}
	for _, tt := range tests {
		if got := fileSlug(tt.filename); got != tt.want {
			t.Errorf("fileSlug(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}
