// Package prompt generates an archscope-style AI context document for the
// set of fake source files produced by fakecodegen.
package prompt

import (
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"sort"
	"strings"

	"github.com/exey/fakecodegen/internal/parser"
)

// FileInfo describes one generated file.
type FileInfo struct {
	Path  string   // relative path inside the output folder
	Lines int      // total line count
	Decls []string // top-level declaration names (functions)
}

// Config holds the parameters for prompt generation.
type Config struct {
	Lang         string               // "go" | "py" | etc.
	FolderName   string               // base name of the output folder (used as module name)
	Files        []FileInfo           // generated files, in any order
	Contributors []parser.Contributor // if empty, fake contributors are generated
	Rng          *rand.Rand
}

var goTechStack = []string{
	"net/http", "encoding/json", "context", "sync", "io", "os", "log/slog",
	"database/sql", "crypto/tls", "time", "math/rand", "bufio", "strings",
	"Gin", "Echo", "Chi", "Fiber", "GORM", "sqlx", "Cobra", "Viper",
	"Prometheus", "OpenTelemetry", "Zap", "Zerolog", "testify", "gRPC",
	"Protocol Buffers", "Redis", "PostgreSQL", "MongoDB", "Kafka", "NATS",
}

var pyTechStack = []string{
	"asyncio", "dataclasses", "typing", "pathlib", "json", "os", "sys",
	"logging", "datetime", "collections", "functools", "itertools",
	"FastAPI", "Flask", "Django", "SQLAlchemy", "Pydantic", "Celery",
	"pytest", "aiohttp", "httpx", "Redis", "PostgreSQL", "MongoDB",
	"NumPy", "pandas", "Prometheus", "OpenTelemetry", "Kafka", "boto3",
}

var goFakeAuthors = []string{
	"Alice Chen", "Bob Kim", "Carlos Ruiz", "Diana Lee", "Ethan Park",
	"Fiona Wang", "Grace Liu", "Henry Zhang", "Iris Nakamura", "Jake Brown",
}

var pyFakeAuthors = []string{
	"Anna Kovac", "Ben Torres", "Clara Singh", "David Müller", "Elena Petrov",
	"Felix Okafor", "Gina Yamamoto", "Hassan Ali", "Isla Campbell", "Jan Nowak",
}

var commitPrefixes = []string{
	"feat", "fix", "refactor", "chore", "docs", "test", "perf", "style", "ci",
}

var staleTaskNames = []string{
	"task", "feature", "bugfix", "hotfix", "release", "chore",
}

var peakDays = []string{"Mon", "Tue", "Wed", "Thu", "Fri"}

func langLabel(lang string) string {
	switch lang {
	case "go":
		return "Go"
	case "py":
		return "Python"
	}
	return lang
}

func pickN(rng *rand.Rand, items []string, n int) []string {
	if n >= len(items) {
		return items
	}
	picked := make([]string, len(items))
	copy(picked, items)
	rng.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
	return picked[:n]
}

// buildContributors returns the contributor list to use: either the ones
// parsed from the source prompt (Contributors in cfg) or freshly generated.
func buildContributors(cfg Config) []parser.Contributor {
	if len(cfg.Contributors) > 0 {
		return cfg.Contributors
	}
	var authorPool []string
	switch cfg.Lang {
	case "go":
		authorPool = goFakeAuthors
	default:
		authorPool = pyFakeAuthors
	}
	count := cfg.Rng.IntN(4) + 3
	authors := pickN(cfg.Rng, authorPool, count)

	contributors := make([]parser.Contributor, len(authors))
	for i, a := range authors {
		commits := cfg.Rng.IntN(120) + 5
		files := cfg.Rng.IntN(500) + 10
		contributors[i] = parser.Contributor{Name: a, Commits: commits, Files: files}
	}
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Commits > contributors[j].Commits
	})
	return contributors
}

// Generate returns the archscope-format prompt document as a string.
func Generate(cfg Config) string {
	label := langLabel(cfg.Lang)

	// Aggregate stats
	totalLines := 0
	totalDecls := 0
	for _, f := range cfg.Files {
		totalLines += f.Lines
		totalDecls += len(f.Decls)
	}

	// Sort files by line count descending for Key Files section
	sorted := make([]FileInfo, len(cfg.Files))
	copy(sorted, cfg.Files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Lines > sorted[j].Lines
	})
	keyFiles := sorted
	if len(keyFiles) > 15 {
		keyFiles = keyFiles[:15]
	}

	// Pick tech stack
	var techPool []string
	switch cfg.Lang {
	case "go":
		techPool = goTechStack
	default:
		techPool = pyTechStack
	}
	techCount := cfg.Rng.IntN(5) + 3
	tech := pickN(cfg.Rng, techPool, techCount)
	sort.Strings(tech)

	contributors := buildContributors(cfg)

	// Total commits = sum of contributor commits
	totalCommits := 0
	for _, c := range contributors {
		totalCommits += c.Commits
	}
	typedCommits := totalCommits * (cfg.Rng.IntN(40) + 60) / 100

	// Hot files: pick top files by random change count
	type hotFile struct {
		path    string
		changes int
	}
	var hotFiles []hotFile
	for _, f := range cfg.Files {
		hotFiles = append(hotFiles, hotFile{
			path:    f.Path,
			changes: cfg.Rng.IntN(160) + 1,
		})
	}
	sort.Slice(hotFiles, func(i, j int) bool {
		return hotFiles[i].changes > hotFiles[j].changes
	})
	if len(hotFiles) > 15 {
		hotFiles = hotFiles[:15]
	}

	// Releases / tags
	tagCount := cfg.Rng.IntN(80) + 20
	latestMajor := cfg.Rng.IntN(2)
	latestMinor := cfg.Rng.IntN(8)
	latestPatch := cfg.Rng.IntN(10) + 1
	latestTag := fmt.Sprintf("%d.%d.%d", latestMajor, latestMinor, latestPatch)
	semverCount := tagCount - cfg.Rng.IntN(5)
	if semverCount < 0 {
		semverCount = tagCount
	}

	// Stale branches
	branchTotal := cfg.Rng.IntN(4) + 2
	staleBranchCount := cfg.Rng.IntN(branchTotal) + 1
	if staleBranchCount > branchTotal-1 {
		staleBranchCount = branchTotal - 1
	}

	var b strings.Builder

	fmt.Fprintf(&b, "# %s — Architecture Context\n\n", label)
	b.WriteString("This document was generated by fakecodegen. ")
	b.WriteString("Use it as context when asking an AI assistant questions about this codebase.\n\n")

	b.WriteString("## Overview\n\n")
	fmt.Fprintf(&b, "- **Platform:** %s\n", label)
	fmt.Fprintf(&b, "- **Files:** %d  |  **Lines:** %s  |  **Declarations:** %d\n",
		len(cfg.Files), fmtNum(totalLines), totalDecls)
	fmt.Fprintf(&b, "- **Modules:** %s\n\n", cfg.FolderName)

	b.WriteString("## Tech Stack\n\n")
	b.WriteString(strings.Join(tech, " · "))
	b.WriteString("\n\n")

	b.WriteString("## Key Files\n\n")
	for _, f := range keyFiles {
		declStr := ""
		if len(f.Decls) > 0 {
			decls := f.Decls
			if len(decls) > 12 {
				decls = decls[:12]
			}
			declStr = " — " + strings.Join(decls, ", ")
		}
		fmt.Fprintf(&b, "- `%s` (%d lines)%s\n",
			filepath.ToSlash(f.Path), f.Lines, declStr)
	}
	b.WriteByte('\n')

	b.WriteString("## Git Stats\n\n")
	fmt.Fprintf(&b, "- **Total commits:** %d  |  **Typed:** %d/%d\n",
		totalCommits, typedCommits, totalCommits)
	b.WriteString("\n")

	// Contributors table
	b.WriteString("### Contributors\n\n")
	b.WriteString("| Author | Commits | Files |\n")
	b.WriteString("|--------|--------:|------:|\n")
	for _, c := range contributors {
		fmt.Fprintf(&b, "| %s | %d | %d |\n", c.Name, c.Commits, c.Files)
	}
	b.WriteByte('\n')

	// File Churn table
	b.WriteString("### File Churn\n\n")
	b.WriteString("| File | Changes |\n")
	b.WriteString("|------|--------:|\n")
	for _, hf := range hotFiles {
		fmt.Fprintf(&b, "| %s | %d |\n", filepath.ToSlash(hf.path), hf.changes)
	}
	b.WriteByte('\n')

	// Releases
	b.WriteString("### Releases\n\n")
	fmt.Fprintf(&b, "**%d tags** · %d semver · latest `%s`\n\n", tagCount, semverCount, latestTag)
	b.WriteString("Tags: ")
	b.WriteString(generateTagList(tagCount, latestMajor, latestMinor, latestPatch, cfg.Rng))
	b.WriteString("\n\n")

	// Branches
	b.WriteString("### Branches\n\n")
	peak := peakDays[cfg.Rng.IntN(len(peakDays))]
	fmt.Fprintf(&b, "**%d total** · peak commit day: %s\n\n", branchTotal, peak)
	if staleBranchCount > 0 {
		b.WriteString("| Stale branch | Days idle |\n")
		b.WriteString("|--------------|----------:|\n")
		for range staleBranchCount {
			prefix := staleTaskNames[cfg.Rng.IntN(len(staleTaskNames))]
			num := cfg.Rng.IntN(900) + 100
			idle := cfg.Rng.IntN(180) + 14
			fmt.Fprintf(&b, "| %s/%d | %d |\n", prefix, num, idle)
		}
		b.WriteByte('\n')
	}

	return b.String()
}

// generateTagList produces a space-separated · list of version tags.
func generateTagList(total, major, minor, patch int, rng *rand.Rand) string {
	type ver struct{ major, minor, patch int }
	var versions []ver

	// Work backwards from the latest tag
	m, n, p := major, minor, patch
	for i := 0; i < total && (m > 0 || n > 0 || p > 0); i++ {
		versions = append(versions, ver{m, n, p})
		if p > 0 {
			p--
		} else if n > 0 {
			n--
			p = rng.IntN(10) + 5
		} else if m > 0 {
			m--
			n = rng.IntN(5) + 3
			p = rng.IntN(10) + 5
		}
	}
	// Reverse so oldest first
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}

	var tags []string
	for _, v := range versions {
		base := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
		// Mix v-prefixed and bare tags
		if rng.IntN(2) == 0 {
			tags = append(tags, "v"+base)
		} else {
			tags = append(tags, base)
		}
	}
	return strings.Join(tags, " · ")
}

func fmtNum(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var out []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(ch))
	}
	return string(out)
}
