# hue-cli

A command-line interface and MCP server for the Philips Hue Bridge. Control your lights, scenes, and rooms from the terminal or through AI agents.

Supports both **local** (LAN) and **remote** (Hue Cloud) access.

## Install

```bash
go install github.com/qselle/hue-cli@latest
```

Or download a binary from [Releases](https://github.com/qselle/hue-cli/releases).

## Setup

### Local mode (same network)

Pair with your Hue Bridge on the local network:

```bash
hue-cli auth
```

This discovers your bridge and asks you to press the link button. If auto-discovery doesn't work:

```bash
hue-cli auth --bridge-ip 192.168.1.42
```

### Remote mode (Hue Cloud)

Control lights from anywhere via the Hue Cloud API:

1. Register a free app at [developers.meethue.com/my-apps](https://developers.meethue.com/my-apps/)
2. Authenticate:

```bash
HUE_CLIENT_ID=xxx HUE_CLIENT_SECRET=xxx HUE_APP_ID=xxx hue-cli auth --remote
```

On headless servers, add `--manual` to paste the authorization code manually.

Tokens auto-refresh transparently. Credentials are stored at `~/.config/hue-cli/config.json`.

## Usage

```bash
# List all lights with status
hue-cli lights

# Control a light
hue-cli lights set "Desk Lamp" --on true
hue-cli lights set "Desk Lamp" --on true --brightness 80
hue-cli lights set "Desk Lamp" --color ff0000
hue-cli lights set "Desk Lamp" --on false

# List and activate scenes
hue-cli scenes
hue-cli scenes activate "Relax"

# List rooms
hue-cli rooms

# JSON output (for scripts and AI agents)
hue-cli lights --json
hue-cli scenes --json

# Check authentication status
hue-cli auth status

# Remove stored credentials
hue-cli auth forget
```

All commands work the same in both local and remote mode — the CLI picks the right transport based on your stored config.

## MCP Server

Exposes five tools: `list_lights`, `set_light`, `list_scenes`, `activate_scene`, and `list_rooms`.

Authenticate first (see [Setup](#setup)), then start the server:

```bash
hue-cli serve                # stdio transport
hue-cli serve --http :8080   # HTTP/SSE transport
```

### Claude Code / Claude Desktop

Add to your MCP config:

```json
{
  "mcpServers": {
    "hue": {
      "command": "hue-cli",
      "args": ["serve"]
    }
  }
}
```

### Available Tools

| Tool | Description |
|------|-------------|
| `list_lights` | List all lights with current state (on/off, brightness, color) |
| `set_light` | Control a light by name — on/off, brightness, color (hex RGB) |
| `list_scenes` | List all available scenes |
| `activate_scene` | Activate a scene by name |
| `list_rooms` | List all rooms with device counts |

## Development

```bash
make build    # build to bin/
make test     # run tests
make lint     # run linter
```

## License

MIT
