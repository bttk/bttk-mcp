package calendarmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bttk/bttk-mcp/pkg/calendar"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	googleCalendar "google.golang.org/api/calendar/v3"
)

var ErrAccessDenied = errors.New("access to calendar is not allowed by configuration")

const defaultCalendarID = "primary"

// AddTools registers Calendar tools to the MCP server.
func AddTools(s *server.MCPServer, client calendar.API, config map[string][]string) {
	s.AddTool(CalendarListTool(), CalendarListHandler(client, config))
	s.AddTool(CalendarListEventsTool(), CalendarListEventsHandler(client, config))
	s.AddTool(CalendarCreateEventTool(), CalendarCreateEventHandler(client, config))
	s.AddTool(CalendarPatchEventTool(), CalendarPatchEventHandler(client, config))
	s.AddTool(CalendarDeleteEventTool(), CalendarDeleteEventHandler(client, config))
	s.AddTool(CalendarMoveEventTool(), CalendarMoveEventHandler(client, config))
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

func checkCalendarAccess(calendarID string, config map[string][]string) error {
	allowedCalendars := config["calendars"]
	if !isCalendarAllowed(calendarID, allowedCalendars) {
		return fmt.Errorf("%w: %s", ErrAccessDenied, calendarID)
	}
	return nil
}

func CalendarListTool() mcp.Tool {
	return mcp.NewTool("calendar_list",
		mcp.WithDescription("List available calendars."),
	)
}

func CalendarListHandler(client calendar.API, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func CalendarListEventsHandler(client calendar.API, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		calendarID := defaultCalendarID
		if val, ok := args["calendar"].(string); ok && val != "" {
			calendarID = val
		}

		if err := checkCalendarAccess(calendarID, config); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
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
		mcp.WithString("recurrence", mcp.Description("Recurrence rules (RRULE) for the event (e.g. ['RRULE:FREQ=DAILY;COUNT=2']).")),
	)
}

func parseRecurrence(recurrenceArg interface{}) ([]string, error) {
	if val, ok := recurrenceArg.(string); ok && val != "" {
		if len(val) > 0 && val[0] == '[' {
			var recurrence []string
			if err := json.Unmarshal([]byte(val), &recurrence); err != nil {
				return nil, err
			}
			return recurrence, nil
		}
		return []string{val}, nil
	}
	if val, ok := recurrenceArg.([]interface{}); ok {
		var recurrence []string
		for _, v := range val {
			if s, ok := v.(string); ok {
				recurrence = append(recurrence, s)
			}
		}
		return recurrence, nil
	}
	return nil, nil
}

func parseEventDateTime(val string) (*googleCalendar.EventDateTime, error) {
	if t, err := time.Parse("2006-01-02", val); err == nil {
		return &googleCalendar.EventDateTime{Date: t.Format("2006-01-02")}, nil
	}
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return nil, err
	}
	return &googleCalendar.EventDateTime{DateTime: t.Format(time.RFC3339)}, nil
}

func CalendarCreateEventHandler(client calendar.API, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		calendarID := defaultCalendarID
		if val, ok := args["calendar"].(string); ok && val != "" {
			calendarID = val
		}

		if err := checkCalendarAccess(calendarID, config); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
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

		recurrence, err := parseRecurrence(args["recurrence"])
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse recurrence: %v", err)), nil
		}

		// Parse times
		start, err := parseEventDateTime(startTimeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid startTime format: %v", err)), nil
		}
		end, err := parseEventDateTime(endTimeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid endTime format: %v", err)), nil
		}

		event := &googleCalendar.Event{
			Summary:     summary,
			Description: description,
			Location:    location,
			Start:       start,
			End:         end,
			Recurrence:  recurrence,
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

func CalendarPatchEventTool() mcp.Tool {
	return mcp.NewTool("calendar_patch_event",
		mcp.WithDescription("Update/Patch an existing event in a specific calendar."),
		mcp.WithString("calendar", mcp.Description("The calendar ID (default: 'primary').")),
		mcp.WithString("eventId", mcp.Required(), mcp.Description("The ID of the event to update.")),
		mcp.WithString("summary", mcp.Description("New title of the event.")),
		mcp.WithString("startTime", mcp.Description("New start time (RFC3339).")),
		mcp.WithString("endTime", mcp.Description("New end time (RFC3339).")),
		mcp.WithString("description", mcp.Description("New description.")),
		mcp.WithString("location", mcp.Description("New location.")),
		mcp.WithString("recurrence", mcp.Description("New recurrence rules (replaces existing).")),
	)
}

func CalendarPatchEventHandler(client calendar.API, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		calendarID := defaultCalendarID
		if val, ok := args["calendar"].(string); ok && val != "" {
			calendarID = val
		}

		if err := checkCalendarAccess(calendarID, config); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		eventID, ok := args["eventId"].(string)
		if !ok {
			return mcp.NewToolResultError("eventId is required"), nil
		}

		event := &googleCalendar.Event{}

		if val, ok := args["summary"].(string); ok {
			event.Summary = val
		}
		if val, ok := args["description"].(string); ok {
			event.Description = val
		}
		if val, ok := args["location"].(string); ok {
			event.Location = val
		}

		if val, ok := args["startTime"].(string); ok && val != "" {
			start, err := parseEventDateTime(val)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid startTime format: %v", err)), nil
			}
			event.Start = start
		}

		if val, ok := args["endTime"].(string); ok && val != "" {
			end, err := parseEventDateTime(val)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid endTime format: %v", err)), nil
			}
			event.End = end
		}

		// Handle Recurrence
		recurrence, err := parseRecurrence(args["recurrence"])
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse recurrence: %v", err)), nil
		}
		if recurrence != nil {
			event.Recurrence = recurrence
		}

		patchedEvent, err := client.PatchEvent(calendarID, eventID, event)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to patch event: %v", err)), nil
		}

		jsonBytes, err := json.Marshal(patchedEvent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal patched event to JSON: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

func CalendarDeleteEventTool() mcp.Tool {
	return mcp.NewTool("calendar_delete_event",
		mcp.WithDescription("Delete an event from a specific calendar."),
		mcp.WithString("calendar", mcp.Description("The calendar ID (default: 'primary').")),
		mcp.WithString("eventId", mcp.Required(), mcp.Description("The ID of the event to delete.")),
	)
}

func CalendarDeleteEventHandler(client calendar.API, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		calendarID := defaultCalendarID
		if val, ok := args["calendar"].(string); ok && val != "" {
			calendarID = val
		}

		if err := checkCalendarAccess(calendarID, config); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		eventID, ok := args["eventId"].(string)
		if !ok {
			return mcp.NewToolResultError("eventId is required"), nil
		}

		if err := client.DeleteEvent(calendarID, eventID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete event: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Event %s deleted successfully from calendar %s", eventID, calendarID)), nil
	}
}

func CalendarMoveEventTool() mcp.Tool {
	return mcp.NewTool("calendar_move_event",
		mcp.WithDescription("Move an event from one calendar to another."),
		mcp.WithString("calendar", mcp.Description("The source calendar ID (default: 'primary').")),
		mcp.WithString("eventId", mcp.Required(), mcp.Description("The ID of the event to move.")),
		mcp.WithString("destination", mcp.Required(), mcp.Description("The destination calendar ID.")),
	)
}

func CalendarMoveEventHandler(client calendar.API, config map[string][]string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		calendarID := defaultCalendarID
		if val, ok := args["calendar"].(string); ok && val != "" {
			calendarID = val
		}

		if err := checkCalendarAccess(calendarID, config); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		eventID, ok := args["eventId"].(string)
		if !ok {
			return mcp.NewToolResultError("eventId is required"), nil
		}

		destinationID, ok := args["destination"].(string)
		if !ok {
			return mcp.NewToolResultError("destination is required"), nil
		}

		if err := checkCalendarAccess(destinationID, config); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("destination calendar: %v", err)), nil
		}

		movedEvent, err := client.MoveEvent(calendarID, eventID, destinationID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to move event: %v", err)), nil
		}

		jsonBytes, err := json.Marshal(movedEvent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal moved event to JSON: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}
