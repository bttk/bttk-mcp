package calendar

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bttk/bttk-mcp/internal/googleapi"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	// ErrReadSecret is returned when the client secret file cannot be read.
	ErrReadSecret = errors.New("unable to read client secret file")
	// ErrParseConfig is returned when the client secret file cannot be parsed.
	ErrParseConfig = errors.New("unable to parse client secret file to config")
	// ErrClientRetrieve is returned when the Calendar client cannot be retrieved.
	ErrClientRetrieve = errors.New("unable to retrieve Calendar client")
	// ErrListCalendars is returned when the calendars cannot be listed.
	ErrListCalendars = errors.New("unable to list calendars")
	// ErrListEvents is returned when the events cannot be listed.
	ErrListEvents = errors.New("unable to retrieve events")
	// ErrCreateEvent is returned when an event cannot be created.
	ErrCreateEvent = errors.New("unable to create event")
	// ErrPatchEvent is returned when an event cannot be patched.
	ErrPatchEvent = errors.New("unable to patch event")
	// ErrDeleteEvent is returned when an event cannot be deleted.
	ErrDeleteEvent = errors.New("unable to delete event")
)

// Client is a wrapper around the Google Calendar API service.
type Client struct {
	Service *calendar.Service
}

// API defines the interface for interacting with Google Calendar.
// This allows for mocking in tests.
type API interface {
	ListCalendars() ([]*calendar.CalendarListEntry, error)
	ListEvents(calendarID string, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error)
	CreateEvent(calendarID string, event *calendar.Event) (*calendar.Event, error)
	PatchEvent(calendarID, eventID string, event *calendar.Event) (*calendar.Event, error)
	DeleteEvent(calendarID, eventID string) error
}

// NewClient creates a new Calendar client.
// It handles the OAuth2 flow if a valid token is not found.
func NewClient(credentialsPath, tokenPath string) (*Client, error) {
	ctx := context.Background()
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadSecret, err)
	}

	client, err := googleapi.GetClient(b, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseConfig, err)
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrClientRetrieve, err)
	}

	return &Client{Service: srv}, nil
}

// ListCalendars lists the available calendars.
func (c *Client) ListCalendars() ([]*calendar.CalendarListEntry, error) {
	list, err := c.Service.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListCalendars, err)
	}
	return list.Items, nil
}

// ListEvents lists upcoming events from the specified calendar.
func (c *Client) ListEvents(calendarID string, timeMin, timeMax string, maxResults int64) ([]*calendar.Event, error) {
	if timeMin == "" {
		timeMin = time.Now().Format(time.RFC3339)
	}
	call := c.Service.Events.List(calendarID).ShowDeleted(false).
		SingleEvents(true).TimeMin(timeMin).OrderBy("startTime")

	if timeMax != "" {
		call = call.TimeMax(timeMax)
	}
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	events, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListEvents, err)
	}
	return events.Items, nil
}

// CreateEvent creates a new event in the specified calendar.
func (c *Client) CreateEvent(calendarID string, event *calendar.Event) (*calendar.Event, error) {
	createdEvent, err := c.Service.Events.Insert(calendarID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateEvent, err)
	}
	return createdEvent, nil
}

// PatchEvent patches an existing event in the specified calendar.
func (c *Client) PatchEvent(calendarID, eventID string, event *calendar.Event) (*calendar.Event, error) {
	patchedEvent, err := c.Service.Events.Patch(calendarID, eventID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPatchEvent, err)
	}
	return patchedEvent, nil
}

// DeleteEvent deletes an event from the specified calendar.
func (c *Client) DeleteEvent(calendarID, eventID string) error {
	err := c.Service.Events.Delete(calendarID, eventID).Do()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDeleteEvent, err)
	}
	return nil
}
