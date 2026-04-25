package main

import (
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Example struct {
	Slug           string
	Title          string
	Description    string
	Sections       []Section
	FullCode       string
	PlaygroundCode string
}

type Section struct {
	ID   string
	Doc  string
	Code string
}

type Group struct {
	Title    string
	Examples []Example
}

type PageData struct {
	Example  Example
	Groups   []Group
	Prev     *Example
	Next     *Example
}

func main() {
	groups, err := loadGroups("examples")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll("docs", 0755); err != nil {
		log.Fatal(err)
	}

	var allExamples []Example
	for _, g := range groups {
		allExamples = append(allExamples, g.Examples...)
	}

	indexTmpl := mustTemplate("templates/index.html")
	exampleTmpl := mustTemplate("templates/example.html")

	writeTemplate("docs/index.html", indexTmpl, map[string]any{"Groups": groups})

	for i, ex := range allExamples {
		data := PageData{Example: ex, Groups: groups}
		if i > 0 {
			p := allExamples[i-1]
			data.Prev = &p
		}
		if i < len(allExamples)-1 {
			n := allExamples[i+1]
			data.Next = &n
		}
		writeTemplate(filepath.Join("docs", ex.Slug+".html"), exampleTmpl, data)
	}

	log.Printf("built %d examples in %d groups → docs/", len(allExamples), len(groups))
}

func loadGroups(dir string) ([]Group, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)

	var groups []Group
	for _, d := range dirs {
		examples, err := loadExamplesFromDir(filepath.Join(dir, d))
		if err != nil {
			return nil, err
		}
		if len(examples) == 0 {
			continue
		}
		groups = append(groups, Group{
			Title:    dirTitle(d),
			Examples: examples,
		})
	}
	return groups, nil
}

func loadExamplesFromDir(dir string) ([]Example, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tengo") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	var examples []Example
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		examples = append(examples, parseExample(f, src))
	}
	return examples, nil
}

func dirTitle(name string) string {
	// Strip leading numeric prefix: "01-basics" → "basics"
	parts := strings.SplitN(name, "-", 2)
	if len(parts) == 2 {
		if _, err := strconv.Atoi(parts[0]); err == nil {
			name = parts[1]
		}
	}
	// Hyphen-separated words → Title Case
	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func parseExample(filename string, src []byte) Example {
	slug := fileSlug(filename)
	normalized := strings.ReplaceAll(string(src), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	type rawBlock struct {
		kind    string
		content string
	}

	var blocks []rawBlock
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			text := strings.TrimPrefix(trimmed, "// ")
			text = strings.TrimPrefix(text, "//")
			if len(blocks) == 0 || blocks[len(blocks)-1].kind != "doc" {
				blocks = append(blocks, rawBlock{kind: "doc", content: text})
			} else {
				blocks[len(blocks)-1].content += "\n" + text
			}
		} else {
			if len(blocks) == 0 || blocks[len(blocks)-1].kind != "code" {
				blocks = append(blocks, rawBlock{kind: "code", content: line})
			} else {
				blocks[len(blocks)-1].content += "\n" + line
			}
		}
	}

	title := slug
	description := ""
	if len(blocks) > 0 && blocks[0].kind == "doc" {
		content := blocks[0].content
		if nl := strings.Index(content, "\n"); nl >= 0 {
			title = strings.TrimPrefix(content[:nl], "# ")
			rest := strings.TrimSpace(content[nl+1:])
			if rest != "" {
				descLines := strings.Split(rest, "\n")
				var parts []string
				for _, l := range descLines {
					l = strings.TrimSpace(l)
					if l == "" {
						break
					}
					parts = append(parts, l)
				}
				description = strings.Join(parts, " ")
			}
			blocks[0].content = rest
			blocks = blocks[1:]
		} else {
			title = strings.TrimPrefix(content, "# ")
			blocks = blocks[1:]
		}
	}

	var sections []Section
	var playgroundCode []string
	i := 0
	for i < len(blocks) {
		var s Section
		if blocks[i].kind == "doc" {
			s.Doc = strings.TrimSpace(blocks[i].content)
			s.ID = slugify(s.Doc)
			i++
			if i < len(blocks) && blocks[i].kind == "code" {
				s.Code = strings.Trim(blocks[i].content, "\n")
				playgroundCode = append(playgroundCode, s.Code)
				i++
			}
		} else {
			s.Code = strings.Trim(blocks[i].content, "\n")
			playgroundCode = append(playgroundCode, s.Code)
			i++
		}
		if s.Doc != "" || s.Code != "" {
			sections = append(sections, s)
		}
	}

	return Example{
		Slug:           slug,
		Title:          title,
		Description:    description,
		Sections:       sections,
		FullCode:       strings.TrimSpace(normalized),
		PlaygroundCode: strings.TrimSpace(strings.Join(playgroundCode, "\n\n")),
	}
}

func fileSlug(filename string) string {
	base := strings.TrimSuffix(filepath.Base(filename), ".tengo")
	parts := strings.SplitN(base, "-", 2)
	if len(parts) == 2 {
		if _, err := strconv.Atoi(parts[0]); err == nil {
			return parts[1]
		}
	}
	return base
}

func slugify(text string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	s := strings.ToLower(lines[0])
	var res strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			res.WriteRune(r)
		} else if r == ' ' || r == '-' {
			res.WriteRune('-')
		}
	}
	return strings.Trim(strings.ReplaceAll(res.String(), "--", "-"), "-")
}

func mustTemplate(path string) *template.Template {
	src, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return template.Must(template.New(filepath.Base(path)).Parse(string(src)))
}

func writeTemplate(path string, tmpl *template.Template, data any) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := tmpl.Execute(f, data); err != nil {
		log.Fatal(err)
	}
	log.Printf("  → %s", path)
}
