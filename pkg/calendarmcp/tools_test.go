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

func (m *MockCalendarAPI) ListCalendars() ([]*googleCalendar.CalendarListEntry, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*googleCalendar.CalendarListEntry), args.Error(1)
}

func (m *MockCalendarAPI) ListEvents(calendarID string, timeMin, timeMax string, maxResults int64) ([]*googleCalendar.Event, error) {
	args := m.Called(calendarID, timeMin, timeMax, maxResults)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*googleCalendar.Event), args.Error(1)
}

func (m *MockCalendarAPI) CreateEvent(calendarID string, event *googleCalendar.Event) (*googleCalendar.Event, error) {
	// For CreateEvent, inspecting the event pointer is tricky for strict equality,
	// so we use mock.MatchedBy or just generic assertion. for simplicity here we assume simple pass-through.
	args := m.Called(calendarID, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*googleCalendar.Event), args.Error(1)
}

func (m *MockCalendarAPI) PatchEvent(calendarID, eventID string, event *googleCalendar.Event) (*googleCalendar.Event, error) {
	args := m.Called(calendarID, eventID, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*googleCalendar.Event), args.Error(1)
}

func (m *MockCalendarAPI) DeleteEvent(calendarID, eventID string) error {
	args := m.Called(calendarID, eventID)
	return args.Error(0)
}

func TestCalendarListTool(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	// Mock Data
	calendars := []*googleCalendar.CalendarListEntry{
		{Id: "cal1", Summary: "Calendar 1", Primary: true},
		{Id: "cal2", Summary: "Calendar 2", Primary: false},
	}
	mockClient.On("ListCalendars").Return(calendars, nil)

	// Config allows all
	config := map[string][]string{}

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
	var resultEntries []*googleCalendar.CalendarListEntry
	err = json.Unmarshal([]byte(textContent.Text), &resultEntries)
	require.NoError(t, err)
	assert.Len(t, resultEntries, 2)
	assert.Equal(t, "cal1", resultEntries[0].Id)
}

func TestCalendarListToolfiltered(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	calendars := []*googleCalendar.CalendarListEntry{
		{Id: "cal1", Summary: "Calendar 1"},
		{Id: "cal2", Summary: "Calendar 2"},
	}
	mockClient.On("ListCalendars").Return(calendars, nil)

	// Config allows only cal1
	config := map[string][]string{
		"calendars": {"cal1"},
	}

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
	var resultEntries []*googleCalendar.CalendarListEntry
	err = json.Unmarshal([]byte(textContent.Text), &resultEntries)
	require.NoError(t, err)
	assert.Len(t, resultEntries, 1)
	assert.Equal(t, "cal1", resultEntries[0].Id)
}

func TestCalendarListEventsTool(t *testing.T) {
	mockClient := new(MockCalendarAPI)

	events := []*googleCalendar.Event{
		{Id: "evt1", Summary: "Event 1"},
	}
	// Note: arguments matching needs to assume zero values for optionals passed as empty string
	mockClient.On("ListEvents", "primary", "", "", int64(0)).Return(events, nil)

	config := map[string][]string{}

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
	var resultEvents []*googleCalendar.Event
	err = json.Unmarshal([]byte(textContent.Text), &resultEvents)
	require.NoError(t, err)
	assert.Len(t, resultEvents, 1)
	assert.Equal(t, "evt1", resultEvents[0].Id)
}

func TestCalendarListEventsTool_Blocked(t *testing.T) {
	mockClient := new(MockCalendarAPI)
	config := map[string][]string{
		"calendars": {"allowed"},
	}

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
	mockClient.On("CreateEvent", "primary", mock.AnythingOfType("*calendar.Event")).Return(expectedEvent, nil)

	config := map[string][]string{}

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
	config := map[string][]string{}

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
	mockClient.On("PatchEvent", "primary", "evt1", mock.AnythingOfType("*calendar.Event")).Return(expectedEvent, nil)

	config := map[string][]string{}

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

	mockClient.On("DeleteEvent", "primary", "evt1").Return(nil)

	config := map[string][]string{}

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
	mockClient.On("CreateEvent", "primary", mock.MatchedBy(func(e *googleCalendar.Event) bool {
		return e.Start.Date == "2023-10-01" && e.End.Date == "2023-10-02" && e.Start.DateTime == "" && e.End.DateTime == ""
	})).Return(expectedEvent, nil)

	config := map[string][]string{}

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

	mockClient.On("PatchEvent", "primary", "evt1", mock.MatchedBy(func(e *googleCalendar.Event) bool {
		return e.Start != nil && e.Start.Date == "2023-10-01" && e.Start.DateTime == ""
	})).Return(expectedEvent, nil)

	config := map[string][]string{}

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
