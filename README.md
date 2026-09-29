# plugin-settings

Runtime configuration for OpenCharly — the externalized `charly settings` CLI.

The plugin owns the command end to end: the `get`/`set`/`list`/`reset`/`path`
grammar and the output formatting. It also owns the entire config subsystem —
reading and writing the runtime config file `~/.config/charly/config.yml` and
resolving the runtime engine — as almost-entirely-pure `kit.LoadRuntimeConfig` /
`SaveRuntimeConfig` file I/O plus validation. The three credential-store touches
(`vnc.password.*` get/set/delete and `secret_backend` reset/name) dispatch
`verb:credential` directly over the reverse channel.

It is a **compiled-in** command plugin (listed in `charly/charly.yml`
`compiled_plugins:`) because its `Invoke(OpRun)` needs the in-proc reverse
channel for those `verb:credential` calls.

## What it provides

| Capability | Surface |
|---|---|
| `command:settings` | the `charly settings` CLI — `get`, `set`, `list`, `reset`, `path` |

## How to use it

Compose the plugin candy in a project, then use the CLI:

```bash
charly settings list
charly settings set engine.podman
charly settings get secret_backend
charly settings path
```

## Layout

- `candy/plugin-settings/` — the plugin module: `command.go` (the CLI grammar),
  `config.go` (the config subsystem), `plugin.go` / `provider.go`,
  `schema/settings.cue`, `cmd/serve/main.go`.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-build:settings` — the runtime configuration surface.
  This candy carries no `skill:` entity of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-build:secrets` — the credential store `settings` reaches for
  `secret_backend`.
- `/charly-internals:plugin` — the plugin/provider model.
