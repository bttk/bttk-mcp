package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client is a wrapper around the Gmail API service.
type Client struct {
	Service *gmail.Service
}

// GmailAPI defines the interface for interacting with Gmail.
// This allows for mocking in tests.
type GmailAPI interface {
	SearchMessages(query string) ([]*gmail.Message, error)
	GetMessage(id string) (*gmail.Message, error)
}

// NewClient creates a new Gmail client.
// It handles the OAuth2 flow if a valid token is not found.
func NewClient(credentialsPath, tokenPath string) (*Client, error) {
	ctx := context.Background()
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	client := getClient(config, tokenPath)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	return &Client{Service: srv}, nil
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
	} else {
		// Token exists, check if it's expired and refresh if necessary
		src := config.TokenSource(context.Background(), tok)
		newTok, err := src.Token()
		if err != nil {
			// If refresh fails, get a new token
			fmt.Printf("Unable to refresh token: %v\n", err)
			tok = getTokenFromWeb(config)
			saveToken(tokenPath, tok)
		} else {
			// If token was refreshed, save it
			if newTok.AccessToken != tok.AccessToken {
				saveToken(tokenPath, newTok)
				tok = newTok
			}
		}
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
				w.Write([]byte("Authentication successful! You can check the terminal now."))
				codeCh <- code
			} else {
				w.Write([]byte("Authentication failed. No code found."))
				codeCh <- ""
			}
		}),
	}

	go server.Serve(l)

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
	json.NewEncoder(f).Encode(token)
}

// SearchMessages searches for messages matching the query.
// It returns a list of simplified message details.
func (c *Client) SearchMessages(query string) ([]*gmail.Message, error) {
	user := "me"
	r, err := c.Service.Users.Messages.List(user).Q(query).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list messages: %v", err)
	}
	return r.Messages, nil
}

// GetMessage retrieves the details of a specific message.
func (c *Client) GetMessage(id string) (*gmail.Message, error) {
	user := "me"
	msg, err := c.Service.Users.Messages.Get(user, id).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get message: %v", err)
	}
	return msg, nil
}
