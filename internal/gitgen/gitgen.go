// Package gitgen creates a real git repository with a fake commit history
// spanning business days from a given start date to today.
package gitgen

import (
	"fmt"
	"math/rand/v2"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/exey/fakecodegen/internal/parser"
)

// Config describes how to build the fake git history.
type Config struct {
	OutputDir    string               // path to the directory that already contains generated files
	StartDate    time.Time            // first commit date (must be a business day or earlier)
	Contributors []parser.Contributor // authors to use; must be non-empty
	FilePaths    []string             // relative file paths inside OutputDir (used in commit messages)
	Rng          *rand.Rand
}

var commitTemplates = []string{
	"feat: add %s support",
	"fix: resolve %s issue",
	"refactor: improve %s",
	"chore: update %s",
	"docs: update %s documentation",
	"test: add tests for %s",
	"perf: optimize %s",
	"style: format %s",
	"fix: handle edge case in %s",
	"feat: implement %s",
	"chore: clean up %s",
	"fix: %s null check",
	"refactor: extract %s helper",
	"feat: expose %s via API",
	"fix: correct %s logic",
	"ci: update %s pipeline",
	"chore: bump %s version",
}

var genericSubjects = []string{
	"auth", "config", "handler", "middleware", "service", "client",
	"database", "cache", "queue", "scheduler", "validator", "parser",
	"encoder", "router", "worker", "logger", "metrics", "session",
	"token", "migration", "seeder", "factory", "repository", "gateway",
}

// Generate initialises a git repo in cfg.OutputDir and creates one commit per
// business day from cfg.StartDate to today.  It uses the git CLI and requires
// git to be installed.
func Generate(cfg Config) error {
	if len(cfg.Contributors) == 0 {
		return fmt.Errorf("gitgen: no contributors provided")
	}

	dir := cfg.OutputDir

	if err := run(dir, "git", "init", "-b", "main"); err != nil {
		// Older git versions don't support -b; fall back
		if err2 := run(dir, "git", "init"); err2 != nil {
			return fmt.Errorf("git init: %w", err2)
		}
	}

	// Suppress detached HEAD advice globally for this repo
	_ = run(dir, "git", "config", "advice.detachedHead", "false")

	days := businessDays(cfg.StartDate, time.Now())
	if len(days) == 0 {
		return fmt.Errorf("gitgen: no business days between %s and today", cfg.StartDate.Format("2006-01-02"))
	}

	// Weighted author pool: more commits → higher chance to be picked
	authorPool := buildAuthorPool(cfg.Contributors)

	// First commit: stage all generated files
	if err := run(dir, "git", "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	author := authorPool[cfg.Rng.IntN(len(authorPool))]
	authorFlag := fmt.Sprintf("%s <%s>", author, nameToEmail(author))
	dateStr := fmtGitDate(days[0], cfg.Rng)

	env := []string{
		"GIT_AUTHOR_DATE=" + dateStr,
		"GIT_COMMITTER_DATE=" + dateStr,
	}
	if err := runEnv(dir, env, "git", "commit",
		"--author="+authorFlag,
		"-m", "feat: initial implementation",
	); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}

	// Subsequent business days: empty commits with realistic messages
	for _, day := range days[1:] {
		author = authorPool[cfg.Rng.IntN(len(authorPool))]
		authorFlag = fmt.Sprintf("%s <%s>", author, nameToEmail(author))
		dateStr = fmtGitDate(day, cfg.Rng)
		msg := randomCommitMsg(cfg.FilePaths, cfg.Rng)

		env = []string{
			"GIT_AUTHOR_DATE=" + dateStr,
			"GIT_COMMITTER_DATE=" + dateStr,
		}
		if err := runEnv(dir, env, "git", "commit",
			"--allow-empty",
			"--author="+authorFlag,
			"-m", msg,
		); err != nil {
			return fmt.Errorf("commit on %s: %w", day.Format("2006-01-02"), err)
		}
	}

	return nil
}

// businessDays returns all Mon–Fri dates in [start, end] inclusive.
func businessDays(start, end time.Time) []time.Time {
	start = truncateToDay(start)
	end = truncateToDay(end)
	var days []time.Time
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			days = append(days, d)
		}
	}
	return days
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// buildAuthorPool creates a slice where each author appears proportionally to
// their commit count, so picking randomly approximates the real distribution.
func buildAuthorPool(contributors []parser.Contributor) []string {
	total := 0
	for _, c := range contributors {
		total += c.Commits
	}
	if total == 0 {
		var names []string
		for _, c := range contributors {
			names = append(names, c.Name)
		}
		return names
	}

	// Cap pool size to avoid huge allocations on large commit counts
	const maxPool = 500
	var pool []string
	for _, c := range contributors {
		slots := c.Commits * maxPool / total
		if slots < 1 {
			slots = 1
		}
		for range slots {
			pool = append(pool, c.Name)
		}
	}
	return pool
}

// nameToEmail derives a lowercase dot-separated email from a display name.
func nameToEmail(name string) string {
	parts := strings.Fields(strings.ToLower(name))
	// Handle Cyrillic or non-ASCII names: transliterate crudely
	var cleaned []string
	for _, p := range parts {
		var sb strings.Builder
		for _, r := range p {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				sb.WriteRune(r)
			}
		}
		if s := sb.String(); s != "" {
			cleaned = append(cleaned, s)
		}
	}
	if len(cleaned) == 0 {
		// Fallback for names that are entirely non-ASCII
		return strings.ReplaceAll(strings.ToLower(name), " ", ".") + "@example.com"
	}
	return strings.Join(cleaned, ".") + "@example.com"
}

// fmtGitDate formats a time as a git-compatible ISO 8601 timestamp with a
// random time-of-day in working hours (9–18).
func fmtGitDate(day time.Time, rng *rand.Rand) string {
	hour := rng.IntN(9) + 9   // 9–17
	min := rng.IntN(60)
	sec := rng.IntN(60)
	t := time.Date(day.Year(), day.Month(), day.Day(), hour, min, sec, 0, time.UTC)
	return t.Format("2006-01-02T15:04:05 +0000")
}

// randomCommitMsg picks a commit message, using a file basename or generic
// subject as the format argument.
func randomCommitMsg(filePaths []string, rng *rand.Rand) string {
	tmpl := commitTemplates[rng.IntN(len(commitTemplates))]
	var subject string
	if len(filePaths) > 0 {
		fp := filePaths[rng.IntN(len(filePaths))]
		base := filepath.Base(fp)
		subject = strings.TrimSuffix(base, filepath.Ext(base))
	} else {
		subject = genericSubjects[rng.IntN(len(genericSubjects))]
	}
	return fmt.Sprintf(tmpl, subject)
}

// run executes a git sub-command in dir, discarding output.
func run(dir string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}

// runEnv is like run but prepends extra environment variables.
func runEnv(dir string, extraEnv []string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}
