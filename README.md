# Bttk's MCP Utils

This repository contains a collection of Model Context Protocol (MCP) servers and utilities designed to empower AI agents with access to personal data and tools.

## Core Components

The project provides the following components:

### MCP Server (`cmd/bttkmcp`)

The combined MCP server merges Gmail, Google Calendar, and Obsidian tools into a single running process. It can run in standard input/output (stdio) mode, or as an HTTP server using Server-Sent Events (SSE).

**Tools:**
*   **Obsidian Tools:**
    *   `obsidian_get_active_file`: Get the content of the active file.
    *   `obsidian_get_daily_note`: Get the content of a daily note.
    *   `obsidian_get_file`: Get the content of a file.
    *   `obsidian_list_files`: List files in the vault.
    *   `obsidian_search_simple`: Simple text search.
    *   `obsidian_search_json_logic`: JSON Logic search.
    *   `obsidian_search_dql`: Dataview Query Language search.
    *   `obsidian_append_active_file`: Append content to the active file.
    *   `obsidian_open_file`: Open a file in Obsidian UI.
*   **Gmail Tools:**
    *   `gmail_search`: Search for messages.
    *   `gmail_read`: Read specific message content by ID.
*   **Calendar Tools:**
    *   `calendar_list`: List available calendars.
    *   `calendar_list_events`: List upcoming events from a specific calendar.
    *   `calendar_create_event`: Create a new event in a specific calendar.

### Obsidian CLI Tool (`cmd/obscom`)

A lightweight command-line interface to interact with Obsidian directly. Currently supports listing all registered Obsidian commands with their Names and IDs.

## Getting Started

### Prerequisites

*   **Go**
*   **Obsidian**: Install the "Local REST API" plugin and generate an API key.
*   **Gmail API**: Requires `credentials.json` from Google Cloud Console (OAuth 2.0 Client ID).
*   **Calendar API**: Requires `credentials.json` from Google Cloud Console (OAuth 2.0 Client ID).

#### Calendar & Gmail API Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project or select an existing one.
3. Enable the [Gmail API](https://console.cloud.google.com/apis/library/gmail.googleapis.com).
4. Create credentials (OAuth 2.0 Client ID):
    *   Create OAuth Client ID: https://console.cloud.google.com/auth/clients
    *   Configure the consent screen (if required).
    *   Set the application type to "Other".
    *   Download the `credentials.json` file.
5. Authenticate using:
    ```bash
    go run ./cmd/bttkmcp auth
    ```

### Installation

```bash
go install github.com/bttk/bttk-mcp/cmd/bttkmcp@latest
```

### Configuration (`config.json`)

Tools are configured via a JSON configuration file (default: `~/.config/bttk-mcp/config.json`).

```json
{
    "obsidian": {
        "enabled": true,
        "url": "https://127.0.0.1:27124",
        "cert": "./obsidian.crt",
        "apikey": "YOUR_OBSIDIAN_API_KEY"
    },
    "gmail": {
        "enabled": true,
        "credentials_file": "./credentials.json",
        "token_file": "./token.json"
    },
    "calendar": {
        "enabled": true,
        "credentials_file": "./credentials.json",
        "token_file": "./token.json",
        "calendars": [
            "primary",
            "example@gmail.com",
            "abcdefghijkl@group.calendar.google.com"
        ]
    },
    "mcp": {
        "address": "localhost:2885",
        "tools": {
            "get_active_file": true,
            "search_simple": true,
            "search_json_logic": true,
            "search_dql": true,
            "get_file": true,
            "list_files": true,
            "open_file": true,
            "gmail_search": true,
            "gmail_read": true
        }
    }
}
```

### Usage with MCP Client

Set the server address in your MCP client configuration (e.g., Gemini CLI `~/.gemini/settings.json`):

```json
{
  "mcpServers": {
    "bttk": {
      "address": "http://localhost:2885/mcp"
    }
  }
}
```

Or in `~/.gemini/antigravity-cli/mcp_config.json`:

```json
{
  "mcpServers": {
    "bttk": {
      "serverURL": "http://localhost:2885/mcp"
    }
  }
}
```

## Deployment as a systemd Service

You can set up `bttkmcp` to run in the background as a systemd user service.

### Installation

To compile `bttkmcp` and copy the binary and the systemd unit file:

```bash
make install
```

This command will:
1. Compile the binaries.
2. Copy the `bttkmcp` binary to `~/bin/`.
3. Copy the `bttkmcp.service` configuration file to `~/.config/systemd/user/` (without overwriting if it already exists).

### Authentication

Verify API authentication for Gmail and Google Calendar before starting the service:

```bash
~/bin/bttkmcp auth
```

### Running as a systemd User Service

To run the combined server as a background service:

```bash
# Reload systemd user configuration
systemctl --user daemon-reload

# Enable and start the service
systemctl --user enable --now bttkmcp.service

# View status
systemctl --user status bttkmcp.service

# Inspect live logs
journalctl --user -u bttkmcp.service -f
```

*(Optional)* Enable user lingering so the service continues running in the background when your terminal/SSH session disconnects:

```bash
loginctl enable-linger $USER
```
