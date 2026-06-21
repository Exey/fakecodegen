// Package gitgen creates a real git repository with a fake commit history
// spanning business days from a given start date to today.
package gitgen

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/exey/fakecodegen/internal/parser"
)

// Config describes how to build the fake git history.
type Config struct {
	OutputDir      string               // path to the directory that already contains generated files
	StartDate      time.Time            // first commit date (must be a business day or earlier)
	EndDate        time.Time            // last commit date (zero = today)
	CommitsPerDay  int                  // average commits per business day (0 or 1 = one per day)
	Contributors   []parser.Contributor // authors to use; must be non-empty
	FilePaths      []string             // relative file paths inside OutputDir (used in commit messages)
	CommitMessages []string             // realistic commit messages to mix into history
	Tags           []string             // version tags to distribute across the commit history
	Rng            *rand.Rand
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

	end := cfg.EndDate
	if end.IsZero() {
		end = time.Now()
	}
	days := businessDays(cfg.StartDate, end)
	if len(days) == 0 {
		return fmt.Errorf("gitgen: no business days between %s and today", cfg.StartDate.Format("2006-01-02"))
	}

	// Weighted author pool: more commits → higher chance to be picked
	authorPool := buildAuthorPool(cfg.Contributors)

	// Pre-generate per-day commit counts and build an author schedule that
	// guarantees every contributor appears at least once.
	cpd := cfg.CommitsPerDay
	if cpd < 1 {
		cpd = 1
	}
	perDay := make([]int, len(days))
	totalSubsequent := 0
	for i := range days {
		// Add ±1 jitter so the history looks organic; minimum 1.
		jitter := cfg.Rng.IntN(3) - 1 // -1, 0, or +1
		perDay[i] = max(1, cpd+jitter)
		if i == 0 {
			totalSubsequent += perDay[i] - 1 // first commit on day 0 is the initial
		} else {
			totalSubsequent += perDay[i]
		}
	}
	authorSchedule := buildAuthorSchedule(cfg.Contributors, authorPool, totalSubsequent, cfg.Rng)

	totalCommits := 1 // initial commit
	for _, n := range perDay {
		totalCommits += n
	}
	totalCommits-- // initial commit already counted in perDay[0]
	fmt.Fprintf(os.Stderr, "git: staging files...\n")

	// First commit on day 0 (earliest slot): stage all generated files.
	if err := run(dir, "git", "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	day0Times := dailyTimestamps(days[0], perDay[0], cfg.Rng)
	firstAuthor := authorPool[cfg.Rng.IntN(len(authorPool))]
	firstFlag := fmt.Sprintf("%s <%s>", firstAuthor, nameToEmail(firstAuthor))
	env := []string{"GIT_AUTHOR_DATE=" + day0Times[0], "GIT_COMMITTER_DATE=" + day0Times[0]}
	if err := runEnv(dir, env, "git", "commit",
		"--author="+firstFlag,
		"-m", "feat: initial implementation",
	); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}
	fmt.Fprintf(os.Stderr, "git: writing %d commits across %d business days...\n", totalCommits, len(days))

	// All subsequent commits: touch 1-3 source files + update CHANGES so every
	// author is credited with real file modifications.
	changelogPath := filepath.Join(dir, "CHANGES")
	schedIdx := 0
	commitsDone := 1 // count the initial commit
	logEvery := max(1, totalCommits/20) // print ~20 progress lines

	doCommit := func(day time.Time, dateStr string) error {
		author := authorSchedule[schedIdx]
		schedIdx++
		authorFlag := fmt.Sprintf("%s <%s>", author, nameToEmail(author))
		msg := randomCommitMsg(cfg.FilePaths, cfg.CommitMessages, cfg.Rng)

		if len(cfg.FilePaths) > 0 {
			for range cfg.Rng.IntN(3) + 1 {
				rel := cfg.FilePaths[cfg.Rng.IntN(len(cfg.FilePaths))]
				full := filepath.Join(dir, filepath.FromSlash(rel))
				if f, err := os.OpenFile(full, os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
					_, _ = f.WriteString("\n")
					f.Close()
					_ = run(dir, "git", "add", filepath.FromSlash(rel))
				}
			}
		}
		entry := fmt.Sprintf("%s %s: %s\n", day.Format("2006-01-02"), author, msg)
		if f, err := os.OpenFile(changelogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			_, _ = f.WriteString(entry)
			f.Close()
			_ = run(dir, "git", "add", "CHANGES")
		}

		env := []string{"GIT_AUTHOR_DATE=" + dateStr, "GIT_COMMITTER_DATE=" + dateStr}
		if err := runEnv(dir, env, "git", "commit",
			"--allow-empty", "--author="+authorFlag, "-m", msg); err != nil {
			return err
		}
		commitsDone++
		if commitsDone%logEvery == 0 || commitsDone == totalCommits {
			fmt.Fprintf(os.Stderr, "git: %d/%d commits (%s)\n",
				commitsDone, totalCommits, day.Format("2006-01-02"))
		}
		return nil
	}

	// Remaining commits on day 0
	for _, ts := range day0Times[1:] {
		if err := doCommit(days[0], ts); err != nil {
			return fmt.Errorf("commit on %s: %w", days[0].Format("2006-01-02"), err)
		}
	}
	// All commits on subsequent days
	for i, day := range days[1:] {
		for _, ts := range dailyTimestamps(day, perDay[i+1], cfg.Rng) {
			if err := doCommit(day, ts); err != nil {
				return fmt.Errorf("commit on %s: %w", day.Format("2006-01-02"), err)
			}
		}
	}

	if len(cfg.Tags) > 0 {
		if err := placeTags(dir, cfg.Tags); err != nil {
			// Non-fatal: tags are cosmetic
			_ = err
		}
	}

	return nil
}

// placeTags distributes version tags across the commit history.
func placeTags(dir string, tags []string) error {
	out, err := exec.Command("git", "-C", dir, "log", "--reverse", "--format=%H").Output()
	if err != nil {
		return err
	}
	hashes := strings.Fields(string(out))
	if len(hashes) == 0 {
		return nil
	}
	step := max(1, len(hashes)/(len(tags)+1))
	for i, tag := range tags {
		idx := (i + 1) * step
		if idx >= len(hashes) {
			idx = len(hashes) - 1
		}
		// Ignore errors for duplicate or invalid tag names
		_ = run(dir, "git", "tag", tag, hashes[idx])
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

// buildAuthorSchedule returns an author name for each of `count` subsequent
// days.  Every contributor gets at least 1 slot (when count >= contributors),
// then remaining slots are filled from the weighted pool.
func buildAuthorSchedule(contributors []parser.Contributor, pool []string, count int, rng *rand.Rand) []string {
	schedule := make([]string, count)

	// Collect unique contributor names (preserve order by commits).
	names := make([]string, 0, len(contributors))
	seen := make(map[string]bool)
	for _, c := range contributors {
		if !seen[c.Name] {
			names = append(names, c.Name)
			seen[c.Name] = true
		}
	}

	// Shuffle to avoid the same ordering each run while still guaranteeing coverage.
	shuffled := append([]string(nil), names...)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	// Evenly space guaranteed slots across the available days.
	if count >= len(shuffled) {
		step := count / len(shuffled)
		if step < 1 {
			step = 1
		}
		for i, name := range shuffled {
			idx := i * step
			if idx >= count {
				idx = count - 1
			}
			schedule[idx] = name
		}
	}

	// Fill remaining empty slots with weighted-random picks.
	for i := range schedule {
		if schedule[i] == "" {
			schedule[i] = pool[rng.IntN(len(pool))]
		}
	}
	return schedule
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
		slots := max(1, c.Commits*maxPool/total)
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

// dailyTimestamps returns n git-format timestamps for the given day, sorted
// chronologically within working hours (9–18 UTC).
func dailyTimestamps(day time.Time, n int, rng *rand.Rand) []string {
	type slot struct{ h, m, s int }
	slots := make([]slot, n)
	for i := range slots {
		slots[i] = slot{rng.IntN(9) + 9, rng.IntN(60), rng.IntN(60)}
	}
	sort.Slice(slots, func(i, j int) bool {
		a, b := slots[i], slots[j]
		if a.h != b.h {
			return a.h < b.h
		}
		if a.m != b.m {
			return a.m < b.m
		}
		return a.s < b.s
	})
	out := make([]string, n)
	for i, s := range slots {
		t := time.Date(day.Year(), day.Month(), day.Day(), s.h, s.m, s.s, 0, time.UTC)
		out[i] = t.Format("2006-01-02T15:04:05 +0000")
	}
	return out
}

// randomCommitMsg picks a commit message. When commitMessages is non-empty,
// 40% of the time it returns one of those directly (realistic messages from
// the ARCHSCOPE prompt); otherwise it generates a templated message.
func randomCommitMsg(filePaths, commitMessages []string, rng *rand.Rand) string {
	if len(commitMessages) > 0 && rng.IntN(10) < 4 {
		return commitMessages[rng.IntN(len(commitMessages))]
	}
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
