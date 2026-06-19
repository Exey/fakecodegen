// Package parser reads an archscope context prompt and extracts the file
// specifications needed to reconstruct a fake repo.
package parser

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// FileSpec describes one file entry from an archscope prompt.
type FileSpec struct {
	Path  string   // relative path as written in the prompt
	Lines int      // target line count
	Decls []string // declaration/function names listed after "—"
}

// PromptSpec is the parsed result of an archscope context document.
type PromptSpec struct {
	Platform string     // e.g. "Rust", "Go", "Python"
	Module   string     // module/folder name
	Files    []FileSpec
}

// Ext returns the file extension that best represents the platform.
// Falls back to counting extensions in the file list when the platform
// name is not recognised.
func (p *PromptSpec) Ext() string {
	switch strings.ToLower(strings.TrimSpace(p.Platform)) {
	case "go":
		return "go"
	case "python":
		return "py"
	case "rust":
		return "rs"
	case "typescript":
		return "ts"
	case "javascript":
		return "js"
	case "c++", "cpp":
		return "cpp"
	default:
		counts := map[string]int{}
		for _, f := range p.Files {
			ext := strings.TrimPrefix(filepath.Ext(f.Path), ".")
			if ext != "" {
				counts[ext]++
			}
		}
		best, bestN := "", 0
		for ext, n := range counts {
			if n > bestN {
				best, bestN = ext, n
			}
		}
		return best
	}
}

// keyFileRe matches lines like:
//
//	- `path/to/file.rs` (1012 lines) — decl1, decl2, decl3
var keyFileRe = regexp.MustCompile("^-\\s+`([^`]+)`\\s+\\((\\d+)\\s+lines?\\)(?:\\s+—\\s+(.+))?$")

// Parse reads an archscope-format context document and extracts file specs.
func Parse(content string) (*PromptSpec, error) {
	spec := &PromptSpec{}
	inKeyFiles := false

	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimRight(raw, " \t")

		// Section headers
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimPrefix(line, "## ")
			inKeyFiles = strings.EqualFold(strings.TrimSpace(heading), "key files")
			continue
		}

		// Platform: `- **Platform:** Rust`
		if strings.Contains(line, "**Platform:**") {
			parts := strings.SplitN(line, "**Platform:**", 2)
			if len(parts) == 2 {
				spec.Platform = strings.TrimSpace(parts[1])
			}
			continue
		}

		// Module: `- **Modules:** sloppify`
		if strings.Contains(line, "**Modules:**") {
			parts := strings.SplitN(line, "**Modules:**", 2)
			if len(parts) == 2 {
				spec.Module = strings.TrimSpace(parts[1])
			}
			continue
		}

		// Key-file entry
		if inKeyFiles {
			if m := keyFileRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				lineCount, _ := strconv.Atoi(m[2])
				var decls []string
				if m[3] != "" {
					for _, d := range strings.Split(m[3], ",") {
						d = strings.TrimSpace(d)
						if d != "" {
							decls = append(decls, d)
						}
					}
				}
				spec.Files = append(spec.Files, FileSpec{
					Path:  m[1],
					Lines: lineCount,
					Decls: decls,
				})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	if len(spec.Files) == 0 {
		return nil, fmt.Errorf("no file entries found — is this a valid archscope prompt with a '## Key Files' section?")
	}
	return spec, nil
}
