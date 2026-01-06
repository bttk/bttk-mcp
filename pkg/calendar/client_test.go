package calendar

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func setupTestServer(t *testing.T) (*httptest.Server, *Client) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me/calendarList":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"items": [{"id": "cal1", "summary": "Calendar 1"}]}`)
		case "/calendars/primary/events":
			if r.Method == "POST" {
				// Create Event
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"id": "evt1", "summary": "New Event", "htmlLink": "http://link"}`)
			} else {
				// List Events
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"items": [{"id": "evt1", "summary": "Event 1"}]}`)
			}
		default:
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	ctx := context.Background()
	srv, err := calendar.NewService(ctx, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
	require.NoError(t, err)

	return ts, &Client{Service: srv}
}

func TestListCalendars(t *testing.T) {
	ts, client := setupTestServer(t)
	defer ts.Close()

	list, err := client.ListCalendars()
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "cal1", list[0].Id)
}

func TestListEvents(t *testing.T) {
	ts, client := setupTestServer(t)
	defer ts.Close()

	// Capture request to verify defaults
	// We need a more complex handler to verify defaults, but for now checking it returns items is a good start.

	events, err := client.ListEvents("primary", "", "", 0)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "evt1", events[0].Id)
}

func TestCreateEvent(t *testing.T) {
	ts, client := setupTestServer(t)
	defer ts.Close()

	event := &calendar.Event{Summary: "New Event"}
	created, err := client.CreateEvent("primary", event)
	require.NoError(t, err)
	assert.Equal(t, "evt1", created.Id)
}

func TestListEvents_Defaults(t *testing.T) {
	// Special server to verify query params
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/calendars/primary/events", r.URL.Path)
		query := r.URL.Query()

		// Verify timeMin default
		timeMin := query.Get("timeMin")
		assert.NotEmpty(t, timeMin, "timeMin should be set by default")
		_, err := time.Parse(time.RFC3339, timeMin)
		assert.NoError(t, err, "timeMin should be RFC3339")

		assert.Equal(t, "true", query.Get("singleEvents"))
		assert.Equal(t, "startTime", query.Get("orderBy"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"items": []}`)
	}))
	defer ts.Close()

	ctx := context.Background()
	srv, err := calendar.NewService(ctx, option.WithEndpoint(ts.URL), option.WithoutAuthentication())
	require.NoError(t, err)
	client := &Client{Service: srv}

	_, err = client.ListEvents("primary", "", "", 0)
	require.NoError(t, err)
}
