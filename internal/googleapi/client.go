package googleapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/term"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
)

var (
	// ErrNonInteractive is returned when interactive authentication is needed but disabled.
	ErrNonInteractive = errors.New("interactive authentication required but disabled (not running in a terminal)")
	// ErrFailedAuthCode is returned when the authentication code cannot be received.
	ErrFailedAuthCode = errors.New("failed to receive auth code")
)

// isInteractive returns true if both stdin and stdout are terminal/TTY devices.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// GetClient handles the OAuth2 flow and returns an authenticated HTTP client.
// It requests scopes for Calendar and Gmail (Read-Only).
func GetClient(ctx context.Context, credentialsJSON []byte, tokenPath string) (*http.Client, error) {
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(credentialsJSON, calendar.CalendarScope, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}
	return getClient(ctx, config, tokenPath)
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(ctx context.Context, config *oauth2.Config, tokenPath string) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		if !isInteractive() {
			return nil, fmt.Errorf("token file %q missing and interactive authentication is disabled: %w", tokenPath, ErrNonInteractive)
		}
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenPath, tok); err != nil {
			log.Printf("Warning: failed to save token: %v", err)
		}
		return config.Client(ctx, tok), nil
	}

	// Token exists, check if it's expired and refresh if necessary
	src := config.TokenSource(ctx, tok)
	newTok, err := src.Token()
	if err != nil {
		// If refresh fails, get a new token
		log.Printf("Unable to refresh token: %v", err)
		if !isInteractive() {
			return nil, fmt.Errorf("failed to refresh token and interactive authentication is disabled: %w", ErrNonInteractive)
		}
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenPath, tok); err != nil {
			log.Printf("Warning: failed to save token: %v", err)
		}
		return config.Client(ctx, tok), nil
	}

	// If token was refreshed, save it
	if newTok.AccessToken != tok.AccessToken {
		if err := saveToken(tokenPath, newTok); err != nil {
			log.Printf("Warning: failed to save token: %v", err)
		}
		tok = newTok
	}
	return config.Client(ctx, tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Create a listener on a random port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("Unable to create listener: %v", err)
		// Fallback to manual copy-paste
		return getTokenFromWebManual(ctx, config)
	}
	defer l.Close()

	// Update the redirect URI to point to our local server
	config.RedirectURL = "http://" + l.Addr().String()

	codeCh := make(chan string)
	errCh := make(chan error, 1)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code != "" {
				_, _ = w.Write([]byte("Authentication successful! You can check the terminal now."))
				codeCh <- code
			} else {
				_, _ = w.Write([]byte("Authentication failed. No code found."))
				codeCh <- ""
			}
		}),
		ReadHeaderTimeout: 10 * time.Second, //nolint:mnd
	}

	go func() {
		if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
			errCh <- err
		}
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Fprintf(os.Stderr, "Opening browser to visit: \n%v\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to open browser: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please open the link manually.")
	}

	// Wait for code or server error
	select {
	case authCode := <-codeCh:
		if authCode == "" {
			return nil, ErrFailedAuthCode
		}
		tok, err := config.Exchange(ctx, authCode)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
		}
		return tok, nil
	case err := <-errCh:
		return nil, fmt.Errorf("auth server error: %w", err)
	}
}

func getTokenFromWebManual(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Fprintf(os.Stderr, "Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	log.Printf("Saving credential file to: %s", path)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("unable to encode token: %w", err)
	}
	return nil
}
