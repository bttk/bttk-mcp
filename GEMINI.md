# Obsidian MCP Agent Project

This project implements a Model Context Protocol (MCP) server for Obsidian, allowing AI models to interact with an Obsidian vault via the [Obsidian Local REST API](https://github.com/vrtmrz/obsidian-local-rest-api) plugin.

## Core Components

### 1. Combined MCP Server (`cmd/bttkmcp`)
The main entry point for the MCP server. It handles:
- **Registration of Tools**: Dynamically enables/disables tools based on JSON configuration.
- **Combined Services**: Merges Obsidian, Gmail, and Google Calendar tools into a single running process.
- **Communication Protocol**: Implements the MCP protocol over Stdio or SSE.
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

### 4. Calendar Client Library (`pkg/calendar`)
A custom Go client for the Google Calendar API.
- **Features**: Authentication handling, event listing, event creation.

### 5. Gmail Client Library (`pkg/gmail`)
A custom Go client for the Gmail API.
- **Features**: Authentication handling, message searching, message reading.

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
| `move_file` | Moves or renames a file in the vault. |
| `open_file` | Opens a specific file in the Obsidian UI. |
| `calendar_list` | Lists available Google Calendars. |
| `calendar_list_events` | Lists upcoming events from a specific calendar. |
| `calendar_create_event` | Creates a new event in a specific calendar. |
| `calendar_move_event` | Moves an event from one calendar to another. |

### MCP structured content

StructuredContent must serialize to a JSON object, not an array.

## Configuration (`config.json`)

The project uses a `config.json` file to specify API credentials and enable/disable specific tools.

```json
{
    "obsidian": {
        "url": "https://127.0.0.1:27124",
        "cert": "obsidian.crt",
        "apikey": "YOUR_API_KEY"
        "apikey": "YOUR_API_KEY"
    },
    "calendar": {
        "enabled": true,
        "calendars": ["primary", "another-calendar-id"]
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
make build
```

### Testing & Linting
```bash
make test
make lint
```

### Running the Combined MCP Server
```bash
go run ./cmd/bttkmcp -config config.json
```

### Running the CLI tool
```bash
go run ./cmd/obscom command list -config config.json
```

## Project Structure
- `cmd/`: Application entry points.
- `pkg/obsidian/`: Core API client implementation.
- `pkg/config/`: Configuration management logic.
- `obsidian.crt`: (Optional) Certificate for secure communication if not using `InsecureSkipVerify`.
