package prompt

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/exey/fakecodegen/internal/parser"
)

// ReadmeConfig holds the information needed to generate a fake project README.
type ReadmeConfig struct {
	Spec       *parser.PromptSpec
	FolderName string // output directory basename — used as project name fallback
}

// GenerateReadme produces a realistic-looking README.md for the fake repo.
// It uses metadata extracted from the archscope prompt so the document
// matches the project being replicated.
func GenerateReadme(cfg ReadmeConfig) string {
	spec := cfg.Spec
	var sb strings.Builder

	// ── Project name ─────────────────────────────────────────────────────────
	projectName := cfg.FolderName
	if projectName == "" || projectName == "." {
		projectName = spec.Module
	}
	if projectName == "" {
		projectName = "project"
	}

	fmt.Fprintf(&sb, "# %s\n\n", projectName)

	// ── One-line description ──────────────────────────────────────────────────
	platform := spec.Platform
	if platform == "" {
		platform = "Go"
	}
	fmt.Fprintf(&sb, "A %s backend platform.\n\n", platform)

	// ── Stats badge line ─────────────────────────────────────────────────────
	if spec.TotalFiles > 0 {
		fmt.Fprintf(&sb, "**%d files** · **%s lines of code**\n\n",
			spec.TotalFiles, formatLines(spec.TotalLines))
	}

	// ── Modules ───────────────────────────────────────────────────────────────
	if spec.Module != "" {
		modules := moduleList(spec)
		if len(modules) > 1 {
			sb.WriteString("## Modules\n\n")
			for _, m := range modules {
				fmt.Fprintf(&sb, "- `%s`\n", m)
			}
			sb.WriteString("\n")
		}
	}

	// ── Tech Stack ────────────────────────────────────────────────────────────
	if len(spec.TechStack) > 0 {
		sb.WriteString("## Tech Stack\n\n")
		sb.WriteString(strings.Join(spec.TechStack, ", "))
		sb.WriteString("\n\n")
	}

	// ── Build ─────────────────────────────────────────────────────────────────
	sb.WriteString("## Build\n\n")
	switch strings.ToLower(platform) {
	case "go":
		sb.WriteString("```bash\ngo mod tidy\ngo build ./...\ngo test ./...\n```\n\n")
	case "python":
		sb.WriteString("```bash\npip install -r requirements.txt\npython -m pytest\n```\n\n")
	case "rust":
		sb.WriteString("```bash\ncargo build --release\ncargo test\n```\n\n")
	case "typescript", "javascript":
		sb.WriteString("```bash\nnpm install\nnpm run build\nnpm test\n```\n\n")
	default:
		sb.WriteString("```bash\ngo mod tidy\ngo build ./...\n```\n\n")
	}

	// ── API Endpoints ─────────────────────────────────────────────────────────
	restRoutes, grpcServices := splitRoutes(spec.InboundRoutes)
	if len(restRoutes) > 0 {
		sb.WriteString("## REST API\n\n")
		for _, r := range restRoutes {
			fmt.Fprintf(&sb, "- `%s`\n", r)
		}
		sb.WriteString("\n")
	}
	if len(grpcServices) > 0 {
		sb.WriteString("## gRPC Services\n\n")
		seen := map[string]bool{}
		for _, svc := range grpcServices {
			if !seen[svc] {
				seen[svc] = true
				fmt.Fprintf(&sb, "- %s\n", svc)
			}
		}
		sb.WriteString("\n")
	}

	// ── Key files snippet ────────────────────────────────────────────────────
	if len(spec.Files) > 0 {
		sb.WriteString("## Key Files\n\n")
		count := 8
		if count > len(spec.Files) {
			count = len(spec.Files)
		}
		for _, f := range spec.Files[:count] {
			dir := filepath.Dir(filepath.ToSlash(f.Path))
			if dir == "." {
				dir = ""
			}
			if dir != "" {
				fmt.Fprintf(&sb, "- [`%s`](%s) — %d lines\n", f.Path, f.Path, f.Lines)
			} else {
				fmt.Fprintf(&sb, "- [`%s`](%s) — %d lines\n", f.Path, f.Path, f.Lines)
			}
		}
		sb.WriteString("\n")
	}

	// ── Contributors ─────────────────────────────────────────────────────────
	if len(spec.Contributors) > 0 {
		sb.WriteString("## Contributors\n\n")
		top := spec.Contributors
		if len(top) > 5 {
			top = top[:5]
		}
		for _, c := range top {
			fmt.Fprintf(&sb, "- %s (%d commits)\n", c.Name, c.Commits)
		}
		sb.WriteString("\n")
	}

	// ── License ───────────────────────────────────────────────────────────────
	sb.WriteString("## License\n\nMIT\n")

	return sb.String()
}

// formatLines pretty-prints a line count (e.g. 99202 → "99 202").
func formatLines(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	// insert thin-space grouping every 3 digits from the right
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, " ")
}

// moduleList returns the individual module names from spec.Module (comma-separated).
func moduleList(spec *parser.PromptSpec) []string {
	// spec.Module currently holds only the first token; try to re-derive from
	// file paths if there's only one module stored.
	dirs := map[string]bool{}
	for _, f := range spec.Files {
		parts := strings.SplitN(filepath.ToSlash(f.Path), "/", 3)
		if len(parts) >= 2 {
			dirs[parts[0]] = true
		}
	}
	if len(dirs) > 1 {
		var list []string
		for d := range dirs {
			list = append(list, d)
		}
		return list
	}
	// Fall back to spec.Module as-is
	return []string{spec.Module}
}

// splitRoutes separates REST route strings from gRPC service names.
func splitRoutes(routes []string) (rest, grpc []string) {
	for _, r := range routes {
		if strings.HasPrefix(r, "/") || strings.Contains(r, " /") ||
			strings.HasPrefix(r, "GET ") || strings.HasPrefix(r, "POST ") ||
			strings.HasPrefix(r, "PUT ") || strings.HasPrefix(r, "DELETE ") ||
			strings.HasPrefix(r, "HEAD ") || strings.HasPrefix(r, "PATCH ") ||
			strings.HasPrefix(r, "HTTP ") {
			rest = append(rest, r)
		} else {
			grpc = append(grpc, r)
		}
	}
	return
}
