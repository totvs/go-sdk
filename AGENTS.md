AGENTS

This file guides agents and contributors about the repository layout and practical
guidelines for working here — especially the `log` package and its facade (abstraction).

Scope
- Applies to the repository root and the entire directory tree below it.

Relevant structure
- `log/` — the `logger` package containing logging utilities.
  - `log/logger.go` — concrete wrapper around `zerolog`.
  - `log/facade.go` — public facade (`LoggerFacade`) and adapter for the internal logger implementation.
  - `log/*.go` — package tests and documentation (`log/logger_test.go`, `log/facade_test.go`, `log/README.md`).
- `examples/` — usage examples (e.g. `examples/logger/main.go`).
- `Makefile` — common targets: `test`, `test-v`, `test-race`, `cover`, `cover-html`, `fmt`, `vet`, `build`, `tidy`, `ci`.
- `README.md` — repository-level documentation.

Development guidelines
- Logging API
  - Prefer the `LoggerFacade` abstraction (in `log/facade.go`) for new code that emits logs to keep callers decoupled
    from a specific implementation (`zerolog`).
  - Quick API summary:
    - Constructors: `NewLog(w, level) -> LoggerFacade`, `NewDefaultLog()`.
  - Context helpers: `ContextWithTrace`, `TraceIDFromContext`, `ContextWithLogger` (stores a `LoggerFacade`), `LoggerFromContext` (returns `LoggerFacade`), and `FromContext`.
    - Field helpers: `WithField`, `WithFields` (available on `LoggerFacade`).
    - Error logging: `LoggerFacade.Error(msg string, err error)` accepts a (possibly nil) `error` to include as the `error` field in the log payload.
      Additionally the facade provides:
      - `Errf(format string, err error, args ...interface{})` — formatted message with error.
      - `Errorw(msg string, err error, fields map[string]interface{})` — message + error + structured fields.
    - Globals/shortcuts: `SetGlobal`, `GetGlobal` and package-level shortcuts `logger.Info/Debug/Warn/Error` and `logger.Infof/...`.

- Adding an adapter for another logging library
  - Add a new file under `log/` (for example `log/<lib>_adapter.go`) implementing `LoggerFacade`.
  - Keep the concrete dependency internal to the `log` package; do not expose it to package consumers.
  - Update `log/README.md` and the root `README.md` when the public API or examples change.

- Tests
  - Place tests next to their code (`log/*.go`) using `_test.go` files.
  - Use `bytes.Buffer` and `httptest` to capture log output and HTTP behaviour; avoid external dependencies.
  - If a test mutates the global logger via `SetGlobal`, restore the previous value with `defer SetGlobal(prev)`.
  - Use `make test` for quick runs and `make test-v` for verbose output.

- Formatting and static analysis
  - Run `gofmt -w .` and `go vet ./...` before submitting changes.
  - Use `make fmt` and `make vet` (Makefile targets are available).

- Patches and automated edits
  - Use the `apply_patch` format when producing automated patches inside this environment.
  - Keep patches focused and small; update docs and examples whenever the public API changes.
  - Do NOT run `git` commands that modify repository state from the agent (for example: `git add`, `git commit`, `git push`, `git reset`, or other write operations).
    Use the `apply_patch` tooling to propose and apply code changes in this environment; leave actual git staging/commits/pushes to a human or CI.

- Build / CI
  - Useful targets: `make build`, `make test`, `make cover`, `make ci`.
- The `ci` target runs `fmt`, `vet`, and `test`.

Concurrency note
- The package-level global logger is stored using `sync/atomic.Value` so `SetGlobal`/`GetGlobal` are safe to call concurrently.
  Prefer setting the global once at startup; runtime swaps are supported but should be used with care.

Best practices
- Fix the root cause rather than applying superficial workarounds.
- Avoid large, unfocused changes; keep patches minimal and targeted.
- Document API changes in `log/README.md` and the root `README.md`.

Notes
- `zerolog` is intentionally encapsulated; the facade was introduced to simplify future migration to other libraries.
- If you need to run `go` commands that touch the module cache or network, notify the environment owner: some environments
  restrict writing to the module cache or disallow network access.

Agent Git Policy

- Do NOT execute any `git` command that modifies repository state (local or remote). This includes but is not limited to:
  - `git add`, `git commit`, `git commit --amend`, `git rebase`, `git merge`, `git reset`, `git rm`, `git clean`, `git tag`, `git push`, `git fetch`, `git pull`, `git checkout` (when creating or switching branches), or any other command that writes or rewrites refs, objects, or remote state.
- Read-only `git` commands (for inspection) are allowed: e.g. `git status` (read-only), `git log`, `git show`, `git diff`, `git ls-files`, `git rev-parse`, `git branch --list`. Prefer `rg`/`sed`/`cat` for file inspection when possible.
- To modify files, use the provided `apply_patch` mechanism. Do not stage, commit or push changes with `git` yourself — leave commits and pushes to a human operator or CI.
- If you believe a `git` write operation is absolutely necessary, explicitly ask for human approval and explain why; do not proceed without consent.
- Do not attempt to hide or rollback your actions with `git` commands; remediation should be handled by a human.

These rules exist to keep repository history and remote state under human control and to prevent accidental or unauthorized changes.
