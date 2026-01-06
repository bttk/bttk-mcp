package calendarmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bttk/bttk-mcp/pkg/calendar"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	googleCalendar "google.golang.org/api/calendar/v3"
)

// AddTools registers Calendar tools to the MCP server.
func AddTools(s *server.MCPServer, client calendar.CalendarAPI, config map[string][]string) {
	s.AddTool(CalendarListTool(), CalendarListHandler(client, config))
	s.AddTool(CalendarListEventsTool(), CalendarListEventsHandler(client, config))
	s.AddTool(CalendarCreateEventTool(), CalendarCreateEventHandler(client, config))
}

func isCalendarAllowed(calendarID string, allowedCalendars []string) bool {
	if len(allowedCalendars) == 0 {
		return true
	}
	for _, c := range allowedCalendars {
		if c == calendarID {
			return true
		}
	}
	return false
}

func CalendarListTool() mcp.Tool {
	return mcp.NewTool("calendar_list",
		mcp.WithDescription("List available calendars."),
	)
}

func CalendarListHandler(client calendar.CalendarAPI, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		calendarList, err := client.ListCalendars()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list calendars: %v", err)), nil
		}

		// Filter list
		var filteredList []*googleCalendar.CalendarListEntry
		allowedCalendars := config["calendars"]

		for _, item := range calendarList {
			if !isCalendarAllowed(item.Id, allowedCalendars) {
				continue
			}
			filteredList = append(filteredList, item)
		}

		if len(filteredList) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		jsonBytes, err := json.Marshal(filteredList)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal calendars to JSON: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

func CalendarListEventsTool() mcp.Tool {
	return mcp.NewTool("calendar_list_events",
		mcp.WithDescription("List upcoming events from a specific calendar."),
		mcp.WithString("calendar", mcp.Description("The calendar ID to list events from (default: 'primary').")),
		mcp.WithString("timeMin", mcp.Description("Lower bound (exclusive) for an event's end time to filter by. RFC3339 format. Default is now.")),
		mcp.WithString("timeMax", mcp.Description("Upper bound (exclusive) for an event's start time to filter by. RFC3339 format.")),
		mcp.WithNumber("maxResults", mcp.Description("Maximum number of events to return.")),
	)
}

func CalendarListEventsHandler(client calendar.CalendarAPI, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		calendarID := "primary"
		if val, ok := args["calendar"].(string); ok && val != "" {
			calendarID = val
		}

		allowedCalendars := config["calendars"]
		if !isCalendarAllowed(calendarID, allowedCalendars) {
			return mcp.NewToolResultError(fmt.Sprintf("access to calendar '%s' is not allowed by configuration", calendarID)), nil
		}

		timeMin := ""
		if val, ok := args["timeMin"].(string); ok {
			timeMin = val
		}
		timeMax := ""
		if val, ok := args["timeMax"].(string); ok {
			timeMax = val
		}
		var maxResults int64
		if val, ok := args["maxResults"].(float64); ok {
			maxResults = int64(val)
		}

		events, err := client.ListEvents(calendarID, timeMin, timeMax, maxResults)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list events: %v", err)), nil
		}

		jsonBytes, err := json.Marshal(events)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal events to JSON: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

func CalendarCreateEventTool() mcp.Tool {
	return mcp.NewTool("calendar_create_event",
		mcp.WithDescription("Create a new event in a specific calendar."),
		mcp.WithString("calendar", mcp.Description("The calendar ID to create the event in (default: 'primary').")),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Title of the event.")),
		mcp.WithString("startTime", mcp.Required(), mcp.Description("Start time of the event (RFC3339 format).")),
		mcp.WithString("endTime", mcp.Required(), mcp.Description("End time of the event (RFC3339 format).")),
		mcp.WithString("description", mcp.Description("Description of the event.")),
		mcp.WithString("location", mcp.Description("Location of the event.")),
	)
}

func CalendarCreateEventHandler(client calendar.CalendarAPI, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		calendarID := "primary"
		if val, ok := args["calendar"].(string); ok && val != "" {
			calendarID = val
		}

		allowedCalendars := config["calendars"]
		if !isCalendarAllowed(calendarID, allowedCalendars) {
			return mcp.NewToolResultError(fmt.Sprintf("access to calendar '%s' is not allowed by configuration", calendarID)), nil
		}

		summary, ok := args["summary"].(string)
		if !ok {
			return mcp.NewToolResultError("summary is required"), nil
		}
		startTimeStr, ok := args["startTime"].(string)
		if !ok {
			return mcp.NewToolResultError("startTime is required"), nil
		}
		endTimeStr, ok := args["endTime"].(string)
		if !ok {
			return mcp.NewToolResultError("endTime is required"), nil
		}
		description, _ := args["description"].(string)
		location, _ := args["location"].(string)

		// Parse times (basic RFC3339 validation)
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid startTime format: %v", err)), nil
		}
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid endTime format: %v", err)), nil
		}

		event := &googleCalendar.Event{
			Summary:     summary,
			Description: description,
			Location:    location,
			Start: &googleCalendar.EventDateTime{
				DateTime: startTime.Format(time.RFC3339),
			},
			End: &googleCalendar.EventDateTime{
				DateTime: endTime.Format(time.RFC3339),
			},
		}

		createdEvent, err := client.CreateEvent(calendarID, event)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create event: %v", err)), nil
		}

		jsonBytes, err := json.Marshal(createdEvent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal created event to JSON: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}
