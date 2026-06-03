package googleapi

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClient_NonInteractive(t *testing.T) {
	// Create a dummy credentials file content (valid JSON with client secret format)
	dummyCreds := []byte(`{
		"installed": {
			"client_id": "dummy-id",
			"project_id": "dummy-project",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token",
			"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
			"client_secret": "dummy-secret",
			"redirect_uris": ["http://localhost"]
		}
	}`)

	// Use a non-existent token path in a temporary directory
	tempDir, err := os.MkdirTemp("", "googleapi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tokenPath := filepath.Join(tempDir, "token.json")

	// Since we are running in tests, stdin/stdout are not terminals,
	// so it should fail with ErrNonInteractive.
	client, err := GetClient(context.Background(), dummyCreds, tokenPath)

	assert.Nil(t, client)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNonInteractive), "expected ErrNonInteractive but got: %v", err)
}
