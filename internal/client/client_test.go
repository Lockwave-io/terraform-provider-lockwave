package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
)

// newTestServer creates an httptest.Server that responds to a single path with the
// given status code and JSON body.
func newTestServer(t *testing.T, method, path string, status int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			t.Errorf("expected method %s, got %s", method, r.Method)
		}
		if r.URL.Path != path {
			t.Errorf("expected path %s, got %s", path, r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Fatal(err)
			}
		}
	}))
}

func TestNewClient_DefaultBaseURL(t *testing.T) {
	c := client.NewClient("", "token", "team-id")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"404 API error", &client.APIError{StatusCode: 404}, true},
		{"403 API error", &client.APIError{StatusCode: 403}, false},
		{"500 API error", &client.APIError{StatusCode: 500}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.IsNotFound(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *client.APIError
		expected string
	}{
		{
			name:     "with message",
			err:      &client.APIError{StatusCode: 422, Message: "validation failed"},
			expected: "lockwave API error (status 422): validation failed",
		},
		{
			name:     "with body only",
			err:      &client.APIError{StatusCode: 500, Body: "internal server error"},
			expected: "lockwave API error (status 500): internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

// ---------- Host tests ----------

func TestCreateHost_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodPost, "/api/v1/hosts", 201, map[string]any{
		"data": map[string]any{
			"id":           "host-uuid-1",
			"display_name": "web-01",
			"hostname":     "web-01.example.com",
			"os":           "linux",
			"arch":         "x86_64",
			"status":       "pending",
			"credential":   "super-secret",
			"created_at":   "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	host, err := c.CreateHost(context.Background(), client.CreateHostRequest{
		DisplayName: "web-01",
		Hostname:    "web-01.example.com",
		OS:          "linux",
		Arch:        "x86_64",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host.ID != "host-uuid-1" {
		t.Errorf("expected ID host-uuid-1, got %s", host.ID)
	}
	if host.Credential != "super-secret" {
		t.Errorf("expected credential super-secret, got %s", host.Credential)
	}
}

func TestGetHost_NotFound(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/hosts/missing-id", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.GetHost(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false for err: %v", err)
	}
}

func TestGetHost_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/hosts/host-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":           "host-uuid-1",
			"display_name": "web-01",
			"hostname":     "web-01.example.com",
			"os":           "linux",
			"arch":         "x86_64",
			"status":       "synced",
			"created_at":   "2024-01-01T00:00:00Z",
			"host_users":   []any{},
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	host, err := c.GetHost(context.Background(), "host-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host.Status != "synced" {
		t.Errorf("expected status synced, got %s", host.Status)
	}
}

func TestUpdateHost_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodPatch, "/api/v1/hosts/host-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":           "host-uuid-1",
			"display_name": "web-01-updated",
			"hostname":     "web-01.example.com",
			"os":           "linux",
			"arch":         "x86_64",
			"status":       "synced",
			"created_at":   "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	host, err := c.UpdateHost(context.Background(), "host-uuid-1", client.UpdateHostRequest{DisplayName: "web-01-updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host.DisplayName != "web-01-updated" {
		t.Errorf("expected display_name web-01-updated, got %s", host.DisplayName)
	}
}

func TestDeleteHost_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	if err := c.DeleteHost(context.Background(), "host-uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteHost_NotFound_NoError(t *testing.T) {
	srv := newTestServer(t, http.MethodDelete, "/api/v1/hosts/gone", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	err := c.DeleteHost(context.Background(), "gone")
	// Client returns the 404 error; callers use IsNotFound to swallow it.
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true")
	}
}

// ---------- SSH Key tests ----------

func TestCreateSshKey_Generate(t *testing.T) {
	keyType := "ed25519"
	srv := newTestServer(t, http.MethodPost, "/api/v1/ssh-keys", 201, map[string]any{
		"data": map[string]any{
			"id":                "key-uuid-1",
			"name":              "deploy-key",
			"fingerprint_sha256": "SHA256:abc",
			"key_type":          "ed25519",
			"private_key":       "-----BEGIN OPENSSH PRIVATE KEY-----",
			"created_at":        "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	key, err := c.CreateSshKey(context.Background(), client.CreateSshKeyRequest{
		Name:    "deploy-key",
		Mode:    "generate",
		KeyType: &keyType,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.ID != "key-uuid-1" {
		t.Errorf("expected ID key-uuid-1, got %s", key.ID)
	}
	if key.PrivateKey == "" {
		t.Error("expected non-empty private key on create")
	}
}

func TestGetSshKey_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/ssh-keys/key-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":                 "key-uuid-1",
			"name":               "deploy-key",
			"fingerprint_sha256": "SHA256:abc",
			"key_type":           "ed25519",
			"blocked_indefinite": false,
			"created_at":         "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	key, err := c.GetSshKey(context.Background(), "key-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.Name != "deploy-key" {
		t.Errorf("expected name deploy-key, got %s", key.Name)
	}
}

// ---------- Assignment tests ----------

func TestCreateAssignment_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodPost, "/api/v1/assignments", 201, map[string]any{
		"data": map[string]any{
			"id": "assignment-uuid-1",
			"ssh_key": map[string]any{
				"id":   "key-uuid-1",
				"name": "deploy-key",
			},
			"host_user": map[string]any{
				"id":      "hu-uuid-1",
				"os_user": "ubuntu",
			},
			"created_at": "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	a, err := c.CreateAssignment(context.Background(), client.CreateAssignmentRequest{
		SshKeyID:   "key-uuid-1",
		HostUserID: "hu-uuid-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != "assignment-uuid-1" {
		t.Errorf("expected ID assignment-uuid-1, got %s", a.ID)
	}
	if a.SshKey.ID != "key-uuid-1" {
		t.Errorf("expected ssh_key.id key-uuid-1, got %s", a.SshKey.ID)
	}
}

func TestGetAssignment_NotFound(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/assignments/missing", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.GetAssignment(context.Background(), "missing")
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got err: %v", err)
	}
}

// ---------- Webhook Endpoint tests ----------

func TestCreateWebhookEndpoint_Success(t *testing.T) {
	description := "My webhook"
	srv := newTestServer(t, http.MethodPost, "/api/v1/webhook-endpoints", 201, map[string]any{
		"data": map[string]any{
			"id":            "wh-uuid-1",
			"url":           "https://example.com/hook",
			"description":   "My webhook",
			"events":        []string{"host.synced"},
			"is_active":     true,
			"failure_count": 0,
			"created_at":    "2024-01-01T00:00:00Z",
			"updated_at":    "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	wh, err := c.CreateWebhookEndpoint(context.Background(), client.CreateWebhookEndpointRequest{
		URL:         "https://example.com/hook",
		Description: &description,
		Events:      []string{"host.synced"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wh.ID != "wh-uuid-1" {
		t.Errorf("expected ID wh-uuid-1, got %s", wh.ID)
	}
	if !wh.IsActive {
		t.Error("expected is_active=true")
	}
}

func TestUpdateWebhookEndpoint_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodPatch, "/api/v1/webhook-endpoints/wh-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":            "wh-uuid-1",
			"url":           "https://example.com/hook-v2",
			"events":        []string{"host.synced", "ssh_key.created"},
			"is_active":     true,
			"failure_count": 0,
			"created_at":    "2024-01-01T00:00:00Z",
			"updated_at":    "2024-01-02T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	wh, err := c.UpdateWebhookEndpoint(context.Background(), "wh-uuid-1", client.UpdateWebhookEndpointRequest{
		URL:    "https://example.com/hook-v2",
		Events: []string{"host.synced", "ssh_key.created"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wh.URL != "https://example.com/hook-v2" {
		t.Errorf("expected url https://example.com/hook-v2, got %s", wh.URL)
	}
}

// ---------- Pagination tests ----------

func TestListHosts_Pagination(t *testing.T) {
	call := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		call++
		if call == 1 {
			// First page — includes a next link pointing to cursor page 2.
			nextURL := srv.URL + "/api/v1/hosts?cursor=page2"
			resp := map[string]any{
				"data": []map[string]any{
					{"id": "h1", "display_name": "host-1", "hostname": "h1.local", "os": "linux", "created_at": "2024-01-01T00:00:00Z"},
				},
				"links": map[string]any{
					"next": nextURL,
				},
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Error(err)
			}
		} else {
			// Second page — no next link.
			resp := map[string]any{
				"data": []map[string]any{
					{"id": "h2", "display_name": "host-2", "hostname": "h2.local", "os": "linux", "created_at": "2024-01-01T00:00:00Z"},
				},
				"links": map[string]any{
					"next": nil,
				},
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Error(err)
			}
		}
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	hosts, err := c.ListHosts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].ID != "h1" || hosts[1].ID != "h2" {
		t.Errorf("unexpected host IDs: %v", []string{hosts[0].ID, hosts[1].ID})
	}
}

// ---------- Error parsing tests ----------

func TestAPIError_ValidationErrors(t *testing.T) {
	srv := newTestServer(t, http.MethodPost, "/api/v1/hosts", 422, map[string]any{
		"message": "The given data was invalid.",
		"errors": map[string][]string{
			"hostname": {"The hostname field is required."},
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.CreateHost(context.Background(), client.CreateHostRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T", err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("expected status 422, got %d", apiErr.StatusCode)
	}
}

func TestGetCurrentTeam_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/teams/team-id", 200, map[string]any{
		"data": map[string]any{
			"id":            "team-id",
			"name":          "Acme Corp",
			"personal_team": false,
			"created_at":    "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	team, err := c.GetCurrentTeam(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if team.Name != "Acme Corp" {
		t.Errorf("expected name Acme Corp, got %s", team.Name)
	}
}

// ---------- Host User tests ----------

func TestCreateHostUser_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodPost, "/api/v1/hosts/host-uuid-1/users", 201, map[string]any{
		"data": map[string]any{
			"id":         "hu-uuid-1",
			"os_user":    "ubuntu",
			"created_at": "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	hu, err := c.CreateHostUser(context.Background(), "host-uuid-1", client.CreateHostUserRequest{
		OsUser: "ubuntu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hu.OsUser != "ubuntu" {
		t.Errorf("expected os_user ubuntu, got %s", hu.OsUser)
	}
}

func TestGetHostUser_NotFound(t *testing.T) {
	// GetHostUser lists and searches; simulate empty list.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data":  []any{},
			"links": map[string]any{"next": nil},
		})
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.GetHostUser(context.Background(), "host-uuid-1", "missing-user-id")
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got err: %v", err)
	}
}
