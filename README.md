# Bttk's MCP Utils

This repository contains a collection of Model Context Protocol (MCP) servers and utilities designed to empower AI agents with access to personal data and tools.

## Core Components

The project currently provides the following MCP servers:

### Obsidian MCP Server (`cmd/obsidianmcp`)

Allows AI agents to interact with an [Obsidian](https://obsidian.md/) vault via the [Obsidian Local REST API](https://github.com/coddingtonbear/obsidian-local-rest-api).

**Tools:**
*   Read-only file access:
    *   `obsidian_get_active_file`: Get the content of the active file.
    *   `obsidian_get_daily_note`: Get the content of a daily note.
    *   `obsidian_get_file`: Get the content of a file.
    *   `obsidian_list_files`: List files in the vault.
*   Search:
    *   `obsidian_search_simple`: Simple text search.
    *   `obsidian_search_json_logic`: JSON Logic search.
    *   `obsidian_search_dql`: Dataview Query Language search.
*   Interactions with the active file:
    *   `obsidian_append_active_file`: Append content to the active file.
    *   `obsidian_open_file`: Open a file in Obsidian UI.

### Gmail MCP Server (`cmd/gmailmcp`)

Provides read-only access to a Gmail account, allowing agents to search and read emails.

**Tools:**
*   `gmail_search`: Search for messages.
*   `gmail_read`: Read specific message content by ID.

### Calendar MCP Server (`cmd/calendarmcp`)

Provides read **and write** access to a Google Calendar account, allowing agents to list calendars and events.

**Tools:**
*   `calendar_list`: List available calendars.
*   `calendar_list_events`: List upcoming events from a specific calendar.
*   `calendar_create_event`: Create a new event in a specific calendar.

#### Listing Calendars

To list available calendars:
```bash
calendarmcp list
```

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
    gmailmcp auth
    ```

### Installation

```bash
go install github.com/bttk/bttk-mcp/cmd/obsidianmcp@latest
go install github.com/bttk/bttk-mcp/cmd/gmailmcp@latest
go install github.com/bttk/bttk-mcp/cmd/calendarmcp@latest
```

### Configuration (`config.json`)

Tools are configured via a JSON configuration file (default: `~/.config/bttk-mcp/config.json`).

```json
{
    "obsidian": {
        "url": "https://127.0.0.1:27124",
        "cert": "./obsidian.crt",
        "apikey": "YOUR_OBSIDIAN_API_KEY"
    },
    "gmail": {
        "credentials_file": "./credentials.json",
        "token_file": "./token.json"
    },
    "calendar": {
        "credentials_file": "./credentials.json",
        "token_file": "./token.json",
        "calendars": [
            "primary",
            "example@gmail.com",
            "abcdefghijkl@group.calendar.google.com"
        ]
    },
    "mcp": {
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

Add the built binaries to your MCP client configuration (e.g., Gemini CLI `settings.json`):

```json
{
  "tools": {
    "allowed": [
      "obsidian_get_active_file",
      "obsidian_search_simple",
      "obsidian_search_json_logic",
      "obsidian_search_dql"
      "obsidian_get_file",
      "obsidian_list_files",
      "calendar_list",
      "calendar_list_events"
    ]
  },
  "mcpServers": {
    "obsidian": {
      "command": "obsidianmcp",
      "trust": false
    },
    "gmail": {
      "command": "gmailmcp",
      "trust": true
    },
    "calendar": {
      "command": "calendarmcp",
      "trust": false
    }
  }
}
```

