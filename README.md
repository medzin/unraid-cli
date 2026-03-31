# Unraid CLI

A command-line client for interacting with the Unraid API.

## Quick Start

1. Add your Unraid server:

```bash
unraid config add tower --url https://192.168.1.100 --api-key YOUR_API_KEY
```

1. List running Docker containers:

```bash
unraid docker list
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

| Platform | Path                                                 |
| -------- | ---------------------------------------------------- |
| Linux    | `~/.config/unraid/config.toml`                       |
| macOS    | `~/Library/Application Support/unraid/config.toml`   |
| Windows  | `C:\Users\<User>\AppData\Roaming\unraid\config.toml` |

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

| Variable         | Description                         |
| ---------------- | ----------------------------------- |
| `UNRAID_URL`     | Server URL                          |
| `UNRAID_API_KEY` | API key for authentication          |
| `UNRAID_SERVER`  | Server name from config file to use |
| `UNRAID_TIMEOUT` | Request timeout in seconds          |

## Commands

### Array

Manage the Unraid storage array.

```bash
# Show array status and disk list
unraid array status

# Start the array
unraid array start

# Stop the array
unraid array stop
```

### Docker

Manage Docker containers on your Unraid server.

```bash
# List running containers
unraid docker list

# List all containers (including stopped)
unraid docker list --all
unraid docker ls -a

# Start a container
unraid docker start <name>

# Stop a container
unraid docker stop <name>

# Restart a container
unraid docker restart <name>

# Pause a container
unraid docker pause <name>

# Unpause a container
unraid docker unpause <name>

# Update a container to the latest image
unraid docker update <name>
```

### Virtual Machines

Manage virtual machines on your Unraid server.

```bash
# List running VMs
unraid vm list

# List all VMs (including stopped)
unraid vm list --all
unraid vm ls -a

# Start a VM
unraid vm start <name>

# Stop a VM (graceful shutdown)
unraid vm stop <name>

# Force stop a VM
unraid vm force-stop <name>

# Pause a VM
unraid vm pause <name>

# Resume a paused VM
unraid vm resume <name>

# Reboot a VM
unraid vm reboot <name>

# Reset a VM (hard reboot)
unraid vm reset <name>
```

### Server Version

Show the Unraid OS and API versions of the connected server.

```bash
unraid server-version
```

### Capabilities

Check which CLI commands are supported by the connected server. Useful when
working with older Unraid versions that may not implement all API operations.

```bash
unraid capabilities

# Against a specific server
unraid --server backup capabilities
```

Example output:

```text
Capabilities for https://192.168.1.100

COMMAND               STATUS
----------------------------------------
array status          supported
array start           supported
array stop            supported
docker list           supported
docker start          supported
docker stop           supported
docker restart        supported
docker pause          supported
docker unpause        supported
docker update         supported
vm list               supported
vm start              supported
vm stop               supported
vm force-stop         not available
vm pause              supported
vm resume             supported
vm reboot             supported
vm reset              supported
```

### CLI Version

```bash
unraid --version
```

### Global Options

These options can be used with any command:

```bash
# Use a specific server from config
unraid --server backup docker list-containers

# Override URL and API key directly
unraid --url https://192.168.1.100 --api-key YOUR_KEY docker list-containers

# Change the request timeout (default is 5 seconds)
unraid --timeout 10 docker list-containers

# Output as JSON (useful for scripting with jq)
unraid --output json docker list
unraid -o json array status
```

Write commands output a result object:

```json
{
  "success": true,
  "message": "Container 'plex' is now running."
}
```

## License

This project is licensed under the MIT License - see the
[LICENSE](LICENSE) file for details.
