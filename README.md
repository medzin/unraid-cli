# Unraid CLI

A command-line client for interacting with the Unraid API.

## Quick Start

1. Add your Unraid server:

```bash
unraid config add tower --url https://192.168.1.100 --api-key YOUR_API_KEY
```

2. List running Docker containers:

```bash
unraid docker list-containers
```

## Configuration

The CLI supports multiple server configurations, allowing you to manage
several Unraid servers from a single client.

### Adding a Server

```bash
unraid config add <name> --url <url> --api-key <api-key>
```

The first server added is automatically set as the default.

### Managing Servers

```bash
# List all configured servers
unraid config list

# Set a different default server
unraid config default <name>

# Remove a server
unraid config remove <name>
```

### Configuration File

Server configurations are stored in a TOML file:

| Platform | Path |
|----------|------|
| Linux | `~/.config/unraid/config.toml` |
| macOS | `~/Library/Application Support/unraid/config.toml` |
| Windows | `C:\Users\<User>\AppData\Roaming\unraid\config.toml` |

Example configuration:

```toml
default = "tower"

[servers.tower]
url = "https://192.168.1.100"
api_key = "your-api-key-here"

[servers.backup]
url = "https://192.168.1.101"
api_key = "another-api-key"
```

### Configuration Priority

The CLI resolves server settings in the following order
(highest priority first):

1. CLI arguments (`--url`, `--api-key`)
2. Environment variables (`UNRAID_URL`, `UNRAID_API_KEY`)
3. Config file (default server or `--server` flag)

## Environment Variables

| Variable | Description |
|----------|-------------|
| `UNRAID_URL` | Server URL |
| `UNRAID_API_KEY` | API key for authentication |
| `UNRAID_SERVER` | Server name from config file to use |

## Commands

### Docker

Manage Docker containers on your Unraid server.

```bash
# List running containers
unraid docker list-containers

# List all containers (including stopped)
unraid docker list-containers --all
unraid docker ls -a
```

### Global Options

These options can be used with any command:

```bash
# Use a specific server from config
unraid --server backup docker list-containers

# Override URL and API key directly
unraid --url https://192.168.1.100 --api-key YOUR_KEY docker list-containers
```

## License

MIT
