package calendarmcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	googleCalendar "google.golang.org/api/calendar/v3"
)

// MockCalendarAPI is a mock implementation of calendar.CalendarAPI
type MockCalendarAPI struct {
	mock.Mock
}

func (m *MockCalendarAPI) ListCalendars(ctx context.Context) ([]*googleCalendar.CalendarListEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*googleCalendar.CalendarListEntry), args.Error(1)
}

func (m *MockCalendarAPI) ListEvents(ctx context.Context, calendarID string, timeMin, timeMax string, maxResults int64) ([]*googleCalendar.Event, error) {
	args := m.Called(ctx, calendarID, timeMin, timeMax, maxResults)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*googleCalendar.Event), args.Error(1)
}

func (m *MockCalendarAPI) CreateEvent(ctx context.Context, calendarID string, event *googleCalendar.Event) (*googleCalendar.Event, error) {
	// For CreateEvent, inspecting the event pointer is tricky for strict equality,
	// so we use mock.MatchedBy or just generic assertion. for simplicity here we assume simple pass-through.
	args := m.Called(ctx, calendarID, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*googleCalendar.Event), args.Error(1)
}

func (m *MockCalendarAPI) PatchEvent(ctx context.Context, calendarID, eventID string, event *googleCalendar.Event) (*googleCalendar.Event, error) {
	args := m.Called(ctx, calendarID, eventID, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*googleCalendar.Event), args.Error(1)
}

func (m *MockCalendarAPI) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	args := m.Called(ctx, calendarID, eventID)
	return args.Error(0)
}

func (m *MockCalendarAPI) MoveEvent(ctx context.Context, calendarID, eventID, destinationID string) (*googleCalendar.Event, error) {
	args := m.Called(ctx, calendarID, eventID, destinationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*googleCalendar.Event), args.Error(1)
}

func TestCalendarListTool(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	// Mock Data
	calendars := []*googleCalendar.CalendarListEntry{
		{Id: "cal1", Summary: "Calendar 1", Primary: true},
		{Id: "cal2", Summary: "Calendar 2", Primary: false},
	}
	mockClient.On("ListCalendars", mock.Anything).Return(calendars, nil)

	// Config allows all
	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarListTool(),
		Handler: CalendarListHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_list",
		},
	})
	require.NoError(t, err)

	assert.False(t, res.IsError)
	assert.NotEmpty(t, res.Content)

	// Verify JSON content
	textContent, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	var result struct {
		Calendars []*googleCalendar.CalendarListEntry `json:"calendars"`
	}
	err = json.Unmarshal([]byte(textContent.Text), &result)
	require.NoError(t, err)
	assert.Len(t, result.Calendars, 2)
	assert.Equal(t, "cal1", result.Calendars[0].Id)
}

func TestCalendarListToolfiltered(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	calendars := []*googleCalendar.CalendarListEntry{
		{Id: "cal1", Summary: "Calendar 1"},
		{Id: "cal2", Summary: "Calendar 2"},
	}
	mockClient.On("ListCalendars", mock.Anything).Return(calendars, nil)

	// Config allows only cal1
	config := NewCalendarConfig([]string{"cal1"})

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarListTool(),
		Handler: CalendarListHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_list",
		},
	})
	require.NoError(t, err)

	assert.False(t, res.IsError)
	textContent, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	var result struct {
		Calendars []*googleCalendar.CalendarListEntry `json:"calendars"`
	}
	err = json.Unmarshal([]byte(textContent.Text), &result)
	require.NoError(t, err)
	assert.Len(t, result.Calendars, 1)
	assert.Equal(t, "cal1", result.Calendars[0].Id)
}

func TestCalendarListEventsTool(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	events := []*googleCalendar.Event{
		{Id: "evt1", Summary: "Event 1"},
	}
	// Note: arguments matching needs to assume zero values for optionals passed as empty string
	mockClient.On("ListEvents", mock.Anything, "primary", "", "", int64(0)).Return(events, nil)

	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarListEventsTool(),
		Handler: CalendarListEventsHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "calendar_list_events",
			Arguments: map[string]interface{}{}, // Use default args
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)

	textContent, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	var result struct {
		Events []*googleCalendar.Event `json:"events"`
	}
	err = json.Unmarshal([]byte(textContent.Text), &result)
	require.NoError(t, err)
	assert.Len(t, result.Events, 1)
	assert.Equal(t, "evt1", result.Events[0].Id)
}

func TestCalendarListEventsTool_Blocked(t *testing.T) {
	mockClient := new(MockCalendarAPI)
	config := NewCalendarConfig([]string{"allowed"})

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarListEventsTool(),
		Handler: CalendarListEventsHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_list_events",
			Arguments: map[string]interface{}{
				"calendar": "blocked",
			},
		},
	})
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "access to calendar is not allowed by configuration: blocked")
}

func TestCalendarCreateEventTool(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	expectedEvent := &googleCalendar.Event{Id: "evt1", HtmlLink: "http://link"}
	mockClient.On("CreateEvent", mock.Anything, "primary", mock.AnythingOfType("*calendar.Event")).Return(expectedEvent, nil)

	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarCreateEventTool(),
		Handler: CalendarCreateEventHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_create_event",
			Arguments: map[string]interface{}{
				"summary":   "My Event",
				"startTime": "2023-10-01T10:00:00Z",
				"endTime":   "2023-10-01T11:00:00Z",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)

	textContent, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	var resultEvent googleCalendar.Event
	err = json.Unmarshal([]byte(textContent.Text), &resultEvent)
	require.NoError(t, err)
	assert.Equal(t, "evt1", resultEvent.Id)
}

func TestCalendarCreateEventTool_MissingArgs(t *testing.T) {
	mockClient := new(MockCalendarAPI)
	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarCreateEventTool(),
		Handler: CalendarCreateEventHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_create_event",
			Arguments: map[string]interface{}{
				"summary": "My Event",
				// Missing startTime, endTime
			},
		},
	})
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "startTime is required")
}

func TestCalendarPatchEventTool(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	expectedEvent := &googleCalendar.Event{Id: "evt1", Summary: "Updated Summary", HtmlLink: "http://link"}

	// We matched against a pointer in CreateEvent, here we do similar for PatchEvent
	mockClient.On("PatchEvent", mock.Anything, "primary", "evt1", mock.AnythingOfType("*calendar.Event")).Return(expectedEvent, nil)

	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarPatchEventTool(),
		Handler: CalendarPatchEventHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_patch_event",
			Arguments: map[string]interface{}{
				"eventId": "evt1",
				"summary": "Updated Summary",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)

	textContent, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	var resultEvent googleCalendar.Event
	err = json.Unmarshal([]byte(textContent.Text), &resultEvent)
	require.NoError(t, err)
	assert.Equal(t, "Updated Summary", resultEvent.Summary)
}

func TestCalendarDeleteEventTool(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	mockClient.On("DeleteEvent", mock.Anything, "primary", "evt1").Return(nil)

	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarDeleteEventTool(),
		Handler: CalendarDeleteEventHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_delete_event",
			Arguments: map[string]interface{}{
				"eventId": "evt1",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)

	textContent, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "deleted successfully")
}

func TestCalendarCreateEventTool_AllDay(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	expectedEvent := &googleCalendar.Event{Id: "evt1"}

	// Expect CreateEvent to be called with Start.Date and End.Date set
	mockClient.On("CreateEvent", mock.Anything, "primary", mock.MatchedBy(func(e *googleCalendar.Event) bool {
		return e.Start.Date == "2023-10-01" && e.End.Date == "2023-10-02" && e.Start.DateTime == "" && e.End.DateTime == ""
	})).Return(expectedEvent, nil)

	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarCreateEventTool(),
		Handler: CalendarCreateEventHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_create_event",
			Arguments: map[string]interface{}{
				"summary":   "All Day Event",
				"startTime": "2023-10-01",
				"endTime":   "2023-10-02",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestCalendarPatchEventTool_AllDay(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	expectedEvent := &googleCalendar.Event{Id: "evt1", Start: &googleCalendar.EventDateTime{Date: "2023-10-01"}}

	mockClient.On("PatchEvent", mock.Anything, "primary", "evt1", mock.MatchedBy(func(e *googleCalendar.Event) bool {
		return e.Start != nil && e.Start.Date == "2023-10-01" && e.Start.DateTime == ""
	})).Return(expectedEvent, nil)

	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarPatchEventTool(),
		Handler: CalendarPatchEventHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_patch_event",
			Arguments: map[string]interface{}{
				"eventId":   "evt1",
				"startTime": "2023-10-01",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestCalendarMoveEventTool(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	expectedEvent := &googleCalendar.Event{Id: "evt1", HtmlLink: "http://link"}
	mockClient.On("MoveEvent", mock.Anything, "primary", "evt1", "destCal").Return(expectedEvent, nil)

	config := NewCalendarConfig(nil)

	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool:    CalendarMoveEventTool(),
		Handler: CalendarMoveEventHandler(mockClient, config),
	})
	require.NoError(t, err)
	defer srv.Close()

	res, err := srv.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calendar_move_event",
			Arguments: map[string]interface{}{
				"eventId":     "evt1",
				"destination": "destCal",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError)

	textContent, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	var resultEvent googleCalendar.Event
	err = json.Unmarshal([]byte(textContent.Text), &resultEvent)
	require.NoError(t, err)
	assert.Equal(t, "evt1", resultEvent.Id)
}
