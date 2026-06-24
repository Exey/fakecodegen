# 🪵 fakecodegen

Generates realistic-looking but completely fake source files for demos, portfolio screenshots, and mockups.  
Can also **reconstruct a fake repo from an [archscope](https://github.com/Exey/archscope) context prompt**, replicating the exact directory structure, file names, declaration names, and approximate line counts described in the document.

## ⚡ Generate Fake Repo in 10 Seconds

```bash
git clone https://github.com/exey/fakecodegen && cd fakecodegen
make install
fakecodegen -from-prompt prompt-Go.md -folder ./fake-repo
```

That's it — `./fake-repo` is now a full fake Go codebase cloned from the archscope prompt, ready for demos or AI context.  
Swap in any archscope `.md` from your own project to reconstruct its structure.

### Add a realistic `.git` history

Pass `-start-date` to create a real git repository with commits on every business day between the given date and today. Commit authors and weights are taken from the `### Contributors` table in the archscope prompt:

```bash
fakecodegen -from-prompt prompt-Go.md -folder ./fake-repo -start-date 2024-01-15
```

Use `-commits-per-day` to control activity density (default `1`). Set it to `10` to mimic a busy team:

```bash
fakecodegen -from-prompt prompt-Go.md -folder ./fake-repo -start-date 2024-01-15 -commits-per-day 10
```

Use `-end-date` to cap the history at a specific date instead of today:

```bash
fakecodegen -from-prompt prompt-Go.md -folder ./fake-repo -start-date 2024-01-15 -end-date 2024-06-01 -commits-per-day 5
```

```text
fake-repo/
├── project-go/
│   ├── auth/…
│   └── services/…
├── README.md          ← auto-generated from prompt metadata
└── .git/              ← full history on business days since 2024-01-15
```

## Install

```bash
go install github.com/exey/fakecodegen@latest
```

Or build from source:

```bash
git clone https://github.com/exey/fakecodegen
cd fakecodegen
go build -o fakecodegen .
```

## Usage

### Generate random fake files

```bash
# 10 Go files in ./output
fakecodegen -lang go -folder ./output -n 10

# 5 Python files
fakecodegen -lang py -folder ./output -n 5

# 8 Rust files plus an ARCHSCOPE.md context prompt
fakecodegen -lang rs -folder ./output -n 8 -prompt

# Swift or Kotlin
fakecodegen -lang swift -folder ./output -n 5
fakecodegen -lang kt -folder ./output -n 5

# TypeScript or JavaScript
fakecodegen -lang ts -folder ./output -n 5
fakecodegen -lang js -folder ./output -n 5
```

### Reconstruct a repo from an archscope prompt

Given an archscope context document (`ARCHSCOPE.md`):

```bash
fakecodegen -from-prompt ARCHSCOPE.md -folder ./fake-repo
```

This reads the `## Key Files` section and for every entry:

- Creates the exact subdirectory structure (`demo/`, `src/render/`, …)
- Generates a fake source file whose functions are named after the declarations listed in the prompt
- Targets approximately the same line count as the original

The language/renderer is detected automatically from each file's extension.  
Pass `-lang` to override all files with a specific renderer:

```bash
# Force Go renderer even though the prompt describes a Rust project
fakecodegen -from-prompt ARCHSCOPE.md -folder ./fake-repo -lang go
```

Add `-prompt` to write a fresh `ARCHSCOPE.md` summarising the generated output:

```bash
fakecodegen -from-prompt ARCHSCOPE.md -folder ./fake-repo -prompt
```

## Flags

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `-lang` | `go` | Renderer to use: `go`, `py`, `rs`, `js`, `ts`. In `-from-prompt` mode auto-detected per file; this flag overrides. |
| `-folder` | `output` | Output directory (created if it does not exist). |
| `-n` | `1` | Number of files to generate (normal mode only). |
| `-prompt` | `false` | Write `ARCHSCOPE.md` in the output folder describing the generated files. |
| `-from-prompt` | — | Path to an archscope context document. Reconstructs the described file tree and writes `README.md`. |
| `-start-date` | — | Generate a `.git` repository with commits from this date to today (format: `2025-01-15`). Contributor names and weights come from the `### Contributors` table in the prompt. |
| `-end-date` | today | End date for the generated git history (format: `2025-06-01`). Defaults to today. |
| `-commits-per-day` | `1` | Average number of commits per business day. Use `10` for a busy-team look. A small ±1 jitter is applied automatically. |

## Supported renderers

| Extension | Output style |
| --------- | ------------ |
| `.go` | `package generated` + top-level `func` declarations + `func init()` for loose statements |
| `.py` | `def` + indented blocks |
| `.rs` | `fn` + brace blocks with `let mut` bindings |
| `.js` | `function` + `var`/`let`/`const` assignments |
| `.ts` | `function(): Type` + typed `let`/`const` assignments |
| `.kt` | `fun(): Type` + typed `val`/`var` assignments |
| `.swift` | `func(): Type` + typed `let`/`var` assignments |
| other | Falls back to the Go renderer |

## How it works

1. An AST of arithmetic, boolean, and control-flow nodes is generated randomly.
2. A language-specific renderer walks the AST and emits source text.
3. Variable and function names are drawn from a curated `names.txt` word list (1 500+ tokens from real codebases).
4. Sloppy comments (`// trust me bro`, `// HACK: this works somehow`, …) are sprinkled in at ~18% probability.

In `-from-prompt` mode the declaration names extracted from the prompt are used directly as function names, and `maxDepth` is tuned per-file so the rendered output lands near the target line count.

## License

MIT
