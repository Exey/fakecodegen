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

// Contributor describes one author from the ### Contributors table.
type Contributor struct {
	Name    string
	Commits int
	Files   int
}

// LongestFunc records a function name and its line count from the
// ## Longest Functions table.  Used to tune per-function generation depth.
type LongestFunc struct {
	Name  string
	Lines int
}

// PromptSpec is the parsed result of an archscope context document.
type PromptSpec struct {
	Platform      string        // e.g. "Rust", "Go", "Python"
	Module        string        // module/folder name (first module from ## Overview)
	TechStack     []string      // tools listed under ## Tech Stack
	TotalFiles    int           // from ## Overview "Files: N"
	TotalLines    int           // from ## Overview "Lines: N"
	Files         []FileSpec
	Contributors  []Contributor  // from ### Contributors table
	LongestFuncs  []LongestFunc  // from ## Longest Functions table
	InboundRoutes []string       // from ## Inbound Traffic bullet list
}

// Ext returns the file extension that best represents the platform.
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

// LongestFuncMap returns a name→lines lookup for quick access during generation.
func (p *PromptSpec) LongestFuncMap() map[string]int {
	m := make(map[string]int, len(p.LongestFuncs))
	for _, lf := range p.LongestFuncs {
		m[lf.Name] = lf.Lines
	}
	return m
}

var keyFileRe = regexp.MustCompile("^-\\s+`([^`]+)`\\s+\\((\\d+)\\s+lines?\\)(?:\\s+—\\s+(.+))?$")

var contributorRowRe = regexp.MustCompile(`^\|\s*([^|]+?)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|`)

// longestFuncRowRe matches lines like: | normalizeDictTitle | 404 | bki (...) |
var longestFuncRowRe = regexp.MustCompile(`^\|\s*([^|]+?)\s*\|\s*(\d+)\s*\|`)

// overviewRe matches "**Files:** 569 | **Lines:** 99202" anywhere in a line.
var overviewFilesRe = regexp.MustCompile(`\*\*Files:\*\*\s*(\d+)`)
var overviewLinesRe = regexp.MustCompile(`\*\*Lines:\*\*\s*(\d+)`)

// inboundRouteRe matches "- [REST] /path" or "- [gRPC] ServiceName".
var inboundRouteRe = regexp.MustCompile(`^-\s+\[(REST|gRPC)\]\s+(.+)$`)

// Parse reads an archscope-format context document and extracts file specs
// and all supplementary metadata sections.
func Parse(content string) (*PromptSpec, error) {
	spec := &PromptSpec{}

	type section int
	const (
		secNone section = iota
		secOverview
		secTechStack
		secKeyFiles
		secContributors
		secLongestFuncs
		secInboundTraffic
	)
	cur := secNone

	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimRight(raw, " \t")

		// ## section headers (level 2)
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			switch strings.ToLower(heading) {
			case "overview":
				cur = secOverview
			case "tech stack":
				cur = secTechStack
			case "key files":
				cur = secKeyFiles
			case "longest functions":
				cur = secLongestFuncs
			case "inbound traffic":
				cur = secInboundTraffic
			default:
				cur = secNone
			}
			continue
		}

		// ### subsection headers (level 3)
		if strings.HasPrefix(line, "### ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "### "))
			if strings.EqualFold(heading, "contributors") {
				cur = secContributors
			} else {
				cur = secNone
			}
			continue
		}

		// Platform: `- **Platform:** Rust` (appears in Overview or as standalone)
		if strings.Contains(line, "**Platform:**") {
			parts := strings.SplitN(line, "**Platform:**", 2)
			if len(parts) == 2 {
				spec.Platform = strings.TrimSpace(parts[1])
			}
			continue
		}

		// Modules: `- **Modules:** api-gw, auth, …`
		if strings.Contains(line, "**Modules:**") {
			parts := strings.SplitN(line, "**Modules:**", 2)
			if len(parts) == 2 {
				// Store first module as canonical module name
				mods := strings.Split(strings.TrimSpace(parts[1]), ",")
				if len(mods) > 0 {
					spec.Module = strings.TrimSpace(mods[0])
				}
			}
			continue
		}

		switch cur {
		case secOverview:
			// Parse "Files: 569 | Lines: 99202" from the Overview bullet
			if m := overviewFilesRe.FindStringSubmatch(line); m != nil {
				spec.TotalFiles, _ = strconv.Atoi(m[1])
			}
			if m := overviewLinesRe.FindStringSubmatch(line); m != nil {
				spec.TotalLines, _ = strconv.Atoi(m[1])
			}

		case secTechStack:
			// Tech stack is a single comma-separated line (or multi-line)
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			for _, t := range strings.Split(trimmed, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					spec.TechStack = append(spec.TechStack, t)
				}
			}

		case secKeyFiles:
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

		case secContributors:
			if strings.HasPrefix(line, "|--") || strings.HasPrefix(line, "| Author") || strings.HasPrefix(line, "| ---") {
				continue
			}
			if m := contributorRowRe.FindStringSubmatch(line); m != nil {
				commits, _ := strconv.Atoi(m[2])
				files, _ := strconv.Atoi(m[3])
				name := strings.TrimSpace(m[1])
				if name != "" && commits > 0 {
					spec.Contributors = append(spec.Contributors, Contributor{
						Name:    name,
						Commits: commits,
						Files:   files,
					})
				}
			}

		case secLongestFuncs:
			// Skip header/separator rows
			if strings.HasPrefix(line, "| Function") || strings.HasPrefix(line, "|--") || strings.HasPrefix(line, "| ---") {
				continue
			}
			if m := longestFuncRowRe.FindStringSubmatch(line); m != nil {
				lines, _ := strconv.Atoi(m[2])
				name := strings.TrimSpace(m[1])
				if name != "" && lines > 0 {
					spec.LongestFuncs = append(spec.LongestFuncs, LongestFunc{Name: name, Lines: lines})
				}
			}

		case secInboundTraffic:
			if m := inboundRouteRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				route := strings.TrimSpace(m[2])
				if route != "" {
					spec.InboundRoutes = append(spec.InboundRoutes, route)
				}
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
