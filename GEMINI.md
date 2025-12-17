# Obsidian MCP Agent Project

This project implements a Model Context Protocol (MCP) server for Obsidian, allowing AI models to interact with an Obsidian vault via the [Obsidian Local REST API](https://github.com/vrtmrz/obsidian-local-rest-api) plugin.

## Core Components

### 1. Obsidian MCP Server (`cmd/obsidianmcp`)
The main entry point for the MCP server. It handles:
- **Registration of Tools**: Dynamically enables/disables tools based on JSON configuration.
- **Communication Protocol**: Implements the MCP protocol over Stdio.
- **Verbose Logging**: Optional logging of incoming/outgoing messages for debugging.

### 2. Obsidian CLI Tool (`cmd/obscom`)
A lightweight command-line interface to interact with Obsidian directly.
- **Command Listing**: Currently supports listing all registered Obsidian commands with their Names and IDs.

### 3. Obsidian Client Library (`pkg/obsidian`)
A custom Go client for the Obsidian Local REST API.
- **Services**: Organizes functionality into logical services:
    - `ActiveFile`: Interactions with the currently open file.
    - `Vault`: File and directory management (CRUD).
    - `Periodic`: Daily, weekly, and monthly note support.
    - `Search`: Simple and JSON Logic-based search.
    - `Commands`: Execute Obsidian commands.
    - `Open`: Open specific files or folders.
- **Configuration**: Managed via `pkg/obsidian/config`.

## Available MCP Tools

| Tool | Description |
| :--- | :--- |
| `get_active_file` | Retrieves the content and metadata of the currently active file. |
| `append_active_file` | Appends text to the end of the active file. |
| `patch_active_file` | Patches the content of the active file (regex replacement). |
| `search_simple` | Performs a simple text search across the vault. |
| `search_json_logic` | Executes complex searches using JSON Logic. |
| `get_daily_note` | Retrieves the content of the today's daily note. |
| `get_file` | Retrieves the content of a specific file by path. |
| `list_files` | Lists files in a specified directory. |
| `create_or_update_file` | Creates a new file or updates an existing one. |
| `open_file` | Opens a specific file in the Obsidian UI. |

## Configuration (`config.json`)

The project uses a `config.json` file to specify API credentials and enable/disable specific tools.

```json
{
    "obsidian": {
        "url": "https://127.0.0.1:27124",
        "cert": "obsidian.crt",
        "apikey": "YOUR_API_KEY"
    },
    "mcp": {
        "tools": {
            "get_active_file": true,
            "search_simple": true,
            "open_file": true
            // ... other tools
        }
    }
}
```

## Setup & Usage

### Prerequisites
1.  **Obsidian**: Install the "Local REST API" plugin.
2.  **API Key**: Obtain an API key from the plugin settings.
3.  **Go**: Ensure Go is installed (1.21+ recommended).

### Building
```bash
go build -o /dev/null ./cmd/obsidianmcp
```

### Running the MCP Server
```bash
go run ./cmd/obsidianmcp -config config.json
```

### Running the CLI tool
```bash
go run ./cmd/obscom command list -config config.json
```

## Project Structure
- `cmd/`: Application entry points.
- `pkg/obsidian/`: Core API client implementation.
- `pkg/obsidian/config/`: Configuration management logic.
- `obsidian.crt`: (Optional) Certificate for secure communication if not using `InsecureSkipVerify`.
