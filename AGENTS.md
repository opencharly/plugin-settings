# AGENTS.md — plugin-settings

Standalone plugin repo for the externalized `charly settings` command
(`command:settings`). The plugin is a Go module at `candy/plugin-settings/`
(module path `github.com/opencharly/plugin-settings/candy/plugin-settings`); the
root `charly.yml` only declares `discover: candy` so the repo is a project and
its candy is scanned.

Canonical files:

- `candy/plugin-settings/charly.yml` — the `plugin-settings:` candy entity
  (`plugin:` block, `plan:` check).
- `candy/plugin-settings/command.go` — the `charly settings` subcommand grammar
  and output formatting.
- `candy/plugin-settings/config.go` — the runtime-config read/write subsystem
  and engine resolution.
- `candy/plugin-settings/schema/settings.cue` — the self-contained schema.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the unified Provider model, command-class dispatch, the per-plugin
  CUE-schema contract, placement. Load before touching the provider or schema.
- `/charly-build:settings` — the runtime configuration surface this command owns.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-settings/` — compile the plugin module.
- `go test ./...` in `candy/plugin-settings/` — the plugin's Go tests
  (`config_test.go`, `schema_serve_test.go`).
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.
- R10 witness: the `charly settings list` end-to-end is exercised by the
  disposable `check-commands-local` bed (see `/charly-check:check`).

## Modify this repo

- Edit the `plugin-settings:` candy entity, the Go source, and
  `schema/settings.cue` **together** — the schema is the single source for the
  `params/` struct, so a field change not mirrored in the schema desyncs the
  generated types.
- The plugin is **compiled-in** and needs the in-proc reverse channel for its
  `verb:credential` calls; it cannot run out-of-process.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
