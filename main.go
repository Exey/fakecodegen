package main

import (
	_ "embed"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/exey/fakecodegen/internal/generator"
	"github.com/exey/fakecodegen/internal/gitgen"
	"github.com/exey/fakecodegen/internal/parser"
	"github.com/exey/fakecodegen/internal/prompt"
	"github.com/exey/fakecodegen/internal/render"
)

//go:embed names.txt
var namesData string

var filePrefixes = []string{
	"main", "app", "index", "server", "client", "utils", "helpers", "config",
	"auth", "database", "db", "models", "routes", "middleware", "handler",
	"controller", "service", "api", "core", "engine", "parser", "lexer",
	"renderer", "manager", "factory", "builder", "adapter", "bridge",
	"observer", "strategy", "command", "state", "proxy", "decorator",
	"facade", "singleton", "prototype", "iterator", "mediator", "visitor",
	"cache", "queue", "stack", "tree", "graph", "node", "schema", "types",
	"constants", "errors", "logger", "metrics", "monitor", "worker",
	"scheduler", "dispatcher", "emitter", "listener", "subscriber",
	"publisher", "producer", "consumer", "validator", "sanitizer",
	"formatter", "transformer", "converter", "encoder", "decoder",
	"serializer", "deserializer", "migrator", "seeder", "fixture",
	"test_utils", "mock", "stub", "setup", "init", "bootstrap", "loader",
	"plugin", "extension", "module", "component", "widget", "layout",
	"view", "template", "style", "theme", "context", "provider", "store",
	"reducer", "action", "effect", "hook", "signal", "stream", "pipeline",
	"filter", "sort", "search", "fetch", "sync", "async_utils", "crypto",
	"hash", "token", "session", "cookie", "storage", "fs_utils", "io",
	"net", "http", "tcp", "udp", "websocket", "grpc", "rpc", "protocol",
}

var fileSuffixes = []string{
	"", "_old", "_new", "_v2", "_backup", "_temp", "_final", "_draft",
	"_helper", "_impl", "_base", "_core", "_internal", "_public",
	"_test", "_spec", "_bench", "_debug",
}

func generateFilenames(n int, ext string, rng *rand.Rand) []string {
	used := make(map[string]bool)
	names := make([]string, 0, n)
	for len(names) < n {
		prefix := filePrefixes[rng.IntN(len(filePrefixes))]
		suffix := fileSuffixes[rng.IntN(len(fileSuffixes))]
		name := fmt.Sprintf("%s%s.%s", prefix, suffix, ext)
		if !used[name] {
			used[name] = true
			names = append(names, name)
		}
	}
	return names
}

func countLines(s string) int { return strings.Count(s, "\n") }

func extForFile(path, langOverride string) string {
	if langOverride != "" {
		return langOverride
	}
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		return "go"
	}
	return ext
}

func usage() {
	fmt.Fprintf(os.Stderr, `fakecodegen — fake source code generator

Usage:
  Generate random files:
    fakecodegen -lang <go|py|rs> -folder <path> [-n <count>] [-prompt] [-start-date 2025-05-01]

  Reconstruct a repo from an archscope prompt:
    fakecodegen -from-prompt <ARCHSCOPE.md> -folder <path> [-lang <ext>] [-start-date 2025-05-01]

Flags:
`)
	flag.PrintDefaults()
}

func parseStartDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func main() {
	lang := flag.String("lang", "", "Language/extension for generated files: go, py, rs, js, ts (default: go; in -from-prompt mode: auto-detected per file)")
	folder := flag.String("folder", "output", "Output directory")
	count := flag.Int("n", 1, "Number of files to generate (normal mode only)")
	promptFlag := flag.Bool("prompt", false, "Write ARCHSCOPE.md context prompt alongside the generated files")
	fromPrompt := flag.String("from-prompt", "", "Path to an archscope context document; reconstruct the described file tree")
	startDate := flag.String("start-date", "", "Generate a fake .git history starting from this date (format: 2025-05-01); only business days get commits")
	flag.Usage = usage
	flag.Parse()

	names := strings.Fields(namesData)
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	outputDir := filepath.Clean(*folder)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create %q: %v\n", outputDir, err)
		os.Exit(1)
	}

	// Parse start date early so we can fail fast before generating files.
	var gitStart time.Time
	var doGit bool
	if *startDate != "" {
		var err error
		gitStart, err = parseStartDate(*startDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: -start-date must be in YYYY-MM-DD format, got %q: %v\n", *startDate, err)
			os.Exit(1)
		}
		doGit = true
	}

	// ── from-prompt mode ──────────────────────────────────────────────────────
	if *fromPrompt != "" {
		data, err := os.ReadFile(*fromPrompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot read %q: %v\n", *fromPrompt, err)
			os.Exit(1)
		}
		spec, err := parser.Parse(string(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// Build a per-function line-count hint map from the ## Longest Functions
		// section so that key functions are generated at the right depth.
		funcLineHints := spec.LongestFuncMap()

		var fileInfos []prompt.FileInfo
		var filePaths []string
		for _, fs := range spec.Files {
			ext := extForFile(fs.Path, *lang)

			state := generator.NewTargeted(fs.Lines, fs.Decls, names, rng)
			program := state.GenerateProgramWithDeclsAndHints(fs.Decls, funcLineHints)

			source, err := render.RenderSourceFile(program, ext, fs.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error rendering %q: %v\n", fs.Path, err)
				os.Exit(1)
			}

			outPath := filepath.Join(outputDir, filepath.FromSlash(fs.Path))
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			if err := os.WriteFile(outPath, []byte(source), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing %q: %v\n", outPath, err)
				os.Exit(1)
			}
			fmt.Println(outPath)

			fileInfos = append(fileInfos, prompt.FileInfo{
				Path:  fs.Path,
				Lines: countLines(source),
				Decls: generator.FunctionNames(program),
			})
			filePaths = append(filePaths, fs.Path)
		}

		// Always write README.md describing the reconstructed project.
		readmeCfg := prompt.ReadmeConfig{
			Spec:       spec,
			FolderName: filepath.Base(outputDir),
		}
		readmeDoc := prompt.GenerateReadme(readmeCfg)
		readmePath := filepath.Join(outputDir, "README.md")
		if err := os.WriteFile(readmePath, []byte(readmeDoc), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write README.md: %v\n", err)
		} else {
			fmt.Println(readmePath)
		}

		if *promptFlag {
			promptLang := spec.Platform
			if *lang != "" {
				promptLang = *lang
			}
			cfg := prompt.Config{
				Lang:         strings.ToLower(promptLang),
				FolderName:   filepath.Base(outputDir),
				Files:        fileInfos,
				Contributors: spec.Contributors,
				Rng:          rng,
			}
			doc := prompt.Generate(cfg)
			promptPath := filepath.Join(outputDir, "ARCHSCOPE.md")
			if err := os.WriteFile(promptPath, []byte(doc), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing prompt: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(promptPath)
		}

		if doGit {
			contributors := spec.Contributors
			if len(contributors) == 0 {
				contributors = defaultContributors()
			}
			gcfg := gitgen.Config{
				OutputDir:    outputDir,
				StartDate:    gitStart,
				Contributors: contributors,
				FilePaths:    filePaths,
				Rng:          rng,
			}
			if err := gitgen.Generate(gcfg); err != nil {
				fmt.Fprintf(os.Stderr, "warning: git history generation failed: %v\n", err)
			} else {
				fmt.Printf("generated .git in %s\n", outputDir)
			}
		}

		fmt.Printf("generated %d files in %s\n", len(spec.Files), outputDir)
		return
	}

	// ── normal generation mode ────────────────────────────────────────────────
	ext := *lang
	if ext == "" {
		ext = "go"
	}
	if ext != "go" && ext != "py" && ext != "rs" && ext != "js" && ext != "ts" {
		fmt.Fprintf(os.Stderr, "error: unsupported -lang %q (use go, py, rs, js, or ts)\n", ext)
		os.Exit(1)
	}
	if *count < 1 {
		fmt.Fprintf(os.Stderr, "error: -n must be >= 1\n")
		os.Exit(1)
	}

	filenames := generateFilenames(*count, ext, rng)
	var fileInfos []prompt.FileInfo
	var filePaths []string

	// Share the generated-name set across all files so that two files in the
	// same Go package never declare the same top-level function name.
	sharedGenerated := make(map[string]bool)

	for _, filename := range filenames {
		state := generator.NewShared(5, sharedGenerated, names, rng)
		program := state.GenerateProgram()

		source, err := render.RenderSourceFile(program, ext, filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		outPath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(outPath, []byte(source), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %q: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Println(outPath)

		fileInfos = append(fileInfos, prompt.FileInfo{
			Path:  filename,
			Lines: countLines(source),
			Decls: generator.FunctionNames(program),
		})
		filePaths = append(filePaths, filename)
	}

	if *promptFlag {
		cfg := prompt.Config{
			Lang:       ext,
			FolderName: filepath.Base(outputDir),
			Files:      fileInfos,
			Rng:        rng,
		}
		doc := prompt.Generate(cfg)
		promptPath := filepath.Join(outputDir, "ARCHSCOPE.md")
		if err := os.WriteFile(promptPath, []byte(doc), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing prompt: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(promptPath)
	}

	if doGit {
		gcfg := gitgen.Config{
			OutputDir:    outputDir,
			StartDate:    gitStart,
			Contributors: defaultContributors(),
			FilePaths:    filePaths,
			Rng:          rng,
		}
		if err := gitgen.Generate(gcfg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git history generation failed: %v\n", err)
		} else {
			fmt.Printf("generated .git in %s\n", outputDir)
		}
	}

	fmt.Printf("generated %d files in %s\n", len(filenames), outputDir)
}

// defaultContributors returns a small set of plausible contributors used when
// no archscope prompt (with a Contributors table) is available.
func defaultContributors() []parser.Contributor {
	return []parser.Contributor{
		{Name: "Alice Chen", Commits: 87, Files: 342},
		{Name: "Bob Kim", Commits: 54, Files: 210},
		{Name: "Carlos Ruiz", Commits: 31, Files: 98},
		{Name: "Diana Lee", Commits: 19, Files: 67},
		{Name: "Ethan Park", Commits: 11, Files: 28},
	}
}
