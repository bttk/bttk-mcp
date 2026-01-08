package googleapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
)

// GetClient handles the OAuth2 flow and returns an authenticated HTTP client.
// It requests scopes for Calendar and Gmail (Read-Only).
func GetClient(credentialsJSON []byte, tokenPath string) (*http.Client, error) {
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(credentialsJSON, calendar.CalendarScope, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}
	return getClient(config, tokenPath), nil
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokenPath string) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenPath, tok)
		return config.Client(context.Background(), tok)
	}

	// Token exists, check if it's expired and refresh if necessary
	src := config.TokenSource(context.Background(), tok)
	newTok, err := src.Token()
	if err != nil {
		// If refresh fails, get a new token
		fmt.Printf("Unable to refresh token: %v\n", err)
		tok = getTokenFromWeb(config)
		saveToken(tokenPath, tok)
		return config.Client(context.Background(), tok)
	}

	// If token was refreshed, save it
	if newTok.AccessToken != tok.AccessToken {
		saveToken(tokenPath, newTok)
		tok = newTok
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	// Create a listener on a random port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("Unable to create listener: %v\n", err)
		// Fallback to manual copy-paste
		return getTokenFromWebManual(config)
	}
	defer l.Close()

	// Update the redirect URI to point to our local server
	config.RedirectURL = "http://" + l.Addr().String()

	codeCh := make(chan string)
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
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Opening browser to visit: \n%v\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Unable to open browser: %v\n", err)
		fmt.Println("Please open the link manually.")
	}

	// Wait for code
	authCode := <-codeCh
	if authCode == "" {
		fmt.Println("Failed to receive auth code.")
		return nil
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		fmt.Printf("Unable to retrieve token from web: %v\n", err)
		return nil
	}
	return tok
}

func getTokenFromWebManual(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		fmt.Printf("Unable to read authorization code: %v\n", err)
		return nil
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		fmt.Printf("Unable to retrieve token from web: %v\n", err)
		return nil
	}
	return tok
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
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		fmt.Printf("Unable to encode token: %v\n", err)
	}
}
