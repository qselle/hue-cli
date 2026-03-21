# hue-cli

Go CLI for the Philips Hue Bridge API (CLIP v2) with optional MCP server support.

## Build & Run

```bash
make build          # builds to bin/hue-cli
make test           # runs all tests
make lint           # runs golangci-lint
make install        # go install to $GOPATH/bin
```

Requires Go 1.22+.

## Architecture

- `main.go` — entry point, calls `cmd.Execute()`
- `cmd/` — Cobra command definitions (thin wiring, calls into `internal/`)
- `internal/auth/` — Bridge discovery, link-button pairing, config storage
- `internal/api/` — Hue Bridge CLIP v2 HTTP client (lights, scenes, rooms)
- `internal/server/` — MCP server (list_lights, set_light, list_scenes, activate_scene, list_rooms)
- `internal/format/` — Shared formatting helpers

## Authentication

1. Run `hue-cli auth` to discover and pair with your Hue Bridge
2. Press the link button on the bridge when prompted
3. Credentials are stored in `~/.config/hue-cli/config.json`

## Conventions

- Follow standard Go project layout (`cmd/` + `internal/`)
- Use Cobra for all CLI commands
- All output commands support `--json` flag for structured output
- Keep commands thin — business logic belongs in `internal/`
