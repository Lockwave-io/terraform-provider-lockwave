package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
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
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data":  []any{},
			"links": map[string]any{"next": nil},
		}); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.GetHostUser(context.Background(), "host-uuid-1", "missing-user-id")
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got err: %v", err)
	}
}

func TestGetHostUser_Success(t *testing.T) {
	akp := "/home/ubuntu/.ssh/authorized_keys"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{
					"id":                   "hu-uuid-1",
					"os_user":              "ubuntu",
					"authorized_keys_path": akp,
					"created_at":           "2024-01-01T00:00:00Z",
				},
			},
			"links": map[string]any{"next": nil},
		}); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	hu, err := c.GetHostUser(context.Background(), "host-uuid-1", "hu-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hu.OsUser != "ubuntu" {
		t.Errorf("expected os_user ubuntu, got %s", hu.OsUser)
	}
	if hu.AuthorizedKeysPath == nil || *hu.AuthorizedKeysPath != akp {
		t.Errorf("unexpected authorized_keys_path: %v", hu.AuthorizedKeysPath)
	}
}

func TestUpdateHostUser_Success(t *testing.T) {
	newPath := "/home/ubuntu/.ssh/authorized_keys"
	srv := newTestServer(t, http.MethodPatch, "/api/v1/hosts/host-uuid-1/users/hu-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":                   "hu-uuid-1",
			"os_user":              "ubuntu",
			"authorized_keys_path": newPath,
			"created_at":           "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	hu, err := c.UpdateHostUser(context.Background(), "host-uuid-1", "hu-uuid-1", client.UpdateHostUserRequest{
		OsUser:             "ubuntu",
		AuthorizedKeysPath: &newPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hu.AuthorizedKeysPath == nil || *hu.AuthorizedKeysPath != newPath {
		t.Errorf("unexpected authorized_keys_path: %v", hu.AuthorizedKeysPath)
	}
}

func TestDeleteHostUser_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	if err := c.DeleteHostUser(context.Background(), "host-uuid-1", "hu-uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------- SSH Key additional tests ----------

func TestUpdateSshKey_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodPatch, "/api/v1/ssh-keys/key-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":                 "key-uuid-1",
			"name":               "deploy-renamed",
			"fingerprint_sha256": "SHA256:abc",
			"key_type":           "ed25519",
			"blocked_indefinite": false,
			"created_at":         "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	key, err := c.UpdateSshKey(context.Background(), "key-uuid-1", client.UpdateSshKeyRequest{Name: "deploy-renamed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.Name != "deploy-renamed" {
		t.Errorf("expected name deploy-renamed, got %s", key.Name)
	}
}

func TestDeleteSshKey_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	if err := c.DeleteSshKey(context.Background(), "key-uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetSshKey_NotFound(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/ssh-keys/missing", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.GetSshKey(context.Background(), "missing")
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got err: %v", err)
	}
}

func TestListSshKeys_Pagination(t *testing.T) {
	call := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		call++
		var resp map[string]any
		if call == 1 {
			nextURL := srv.URL + "/api/v1/ssh-keys?cursor=page2"
			resp = map[string]any{
				"data": []map[string]any{
					{"id": "k1", "name": "key-1", "fingerprint_sha256": "SHA256:a", "key_type": "ed25519", "blocked_indefinite": false, "created_at": "2024-01-01T00:00:00Z"},
				},
				"links": map[string]any{"next": nextURL},
			}
		} else {
			resp = map[string]any{
				"data": []map[string]any{
					{"id": "k2", "name": "key-2", "fingerprint_sha256": "SHA256:b", "key_type": "rsa", "blocked_indefinite": false, "created_at": "2024-01-01T00:00:00Z"},
				},
				"links": map[string]any{"next": nil},
			}
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	keys, err := c.ListSshKeys(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].ID != "k1" || keys[1].ID != "k2" {
		t.Errorf("unexpected key IDs: %v %v", keys[0].ID, keys[1].ID)
	}
}

// ---------- Assignment additional tests ----------

func TestDeleteAssignment_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	if err := c.DeleteAssignment(context.Background(), "assignment-uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAssignment_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/assignments/assignment-uuid-1", 200, map[string]any{
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
	a, err := c.GetAssignment(context.Background(), "assignment-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != "assignment-uuid-1" {
		t.Errorf("expected ID assignment-uuid-1, got %s", a.ID)
	}
	if a.SshKey == nil || a.SshKey.ID != "key-uuid-1" {
		t.Errorf("unexpected ssh_key: %v", a.SshKey)
	}
}

// ---------- Webhook Endpoint additional tests ----------

func TestGetWebhookEndpoint_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/webhook-endpoints/wh-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":            "wh-uuid-1",
			"url":           "https://example.com/hook",
			"events":        []string{"host.synced"},
			"is_active":     true,
			"failure_count": 0,
			"created_at":    "2024-01-01T00:00:00Z",
			"updated_at":    "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	wh, err := c.GetWebhookEndpoint(context.Background(), "wh-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wh.URL != "https://example.com/hook" {
		t.Errorf("expected url https://example.com/hook, got %s", wh.URL)
	}
}

func TestGetWebhookEndpoint_NotFound(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/webhook-endpoints/missing", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.GetWebhookEndpoint(context.Background(), "missing")
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got err: %v", err)
	}
}

func TestDeleteWebhookEndpoint_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	if err := c.DeleteWebhookEndpoint(context.Background(), "wh-uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------- Notification Channel tests ----------

func TestCreateNotificationChannel_Slack(t *testing.T) {
	srv := newTestServer(t, http.MethodPost, "/api/v1/notification-channels", 201, map[string]any{
		"data": map[string]any{
			"id":         "nc-uuid-1",
			"type":       "slack",
			"name":       "Slack Alerts",
			"config":     map[string]any{"webhook_url": "https://hooks.slack.com/services/T0001/B0001/secret"},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	nc, err := c.CreateNotificationChannel(context.Background(), client.CreateNotificationChannelRequest{
		Type: "slack",
		Name: "Slack Alerts",
		Config: client.NotificationChannelConfig{
			"webhook_url": "https://hooks.slack.com/services/T0001/B0001/secret",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nc.ID != "nc-uuid-1" {
		t.Errorf("expected ID nc-uuid-1, got %s", nc.ID)
	}
	if nc.Type != "slack" {
		t.Errorf("expected type slack, got %s", nc.Type)
	}
	if !nc.IsActive {
		t.Error("expected is_active=true")
	}
	if nc.Config["webhook_url"] != "https://hooks.slack.com/services/T0001/B0001/secret" {
		t.Errorf("unexpected webhook_url in config: %v", nc.Config["webhook_url"])
	}
}

func TestCreateNotificationChannel_Email(t *testing.T) {
	srv := newTestServer(t, http.MethodPost, "/api/v1/notification-channels", 201, map[string]any{
		"data": map[string]any{
			"id":         "nc-uuid-2",
			"type":       "email",
			"name":       "Email Alerts",
			"config":     map[string]any{"recipients": []string{"ops@example.com", "sre@example.com"}},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	nc, err := c.CreateNotificationChannel(context.Background(), client.CreateNotificationChannelRequest{
		Type: "email",
		Name: "Email Alerts",
		Config: client.NotificationChannelConfig{
			"recipients": []string{"ops@example.com", "sre@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nc.ID != "nc-uuid-2" {
		t.Errorf("expected ID nc-uuid-2, got %s", nc.ID)
	}
	if nc.Type != "email" {
		t.Errorf("expected type email, got %s", nc.Type)
	}
}

func TestGetNotificationChannel_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/notification-channels/nc-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":         "nc-uuid-1",
			"type":       "slack",
			"name":       "Slack Alerts",
			"config":     map[string]any{"webhook_url": "https://hooks.slack.com/services/T0001/B0001/secret"},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	nc, err := c.GetNotificationChannel(context.Background(), "nc-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nc.Name != "Slack Alerts" {
		t.Errorf("expected name Slack Alerts, got %s", nc.Name)
	}
}

func TestGetNotificationChannel_NotFound(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/notification-channels/missing", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.GetNotificationChannel(context.Background(), "missing")
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got err: %v", err)
	}
}

func TestUpdateNotificationChannel_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodPatch, "/api/v1/notification-channels/nc-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":         "nc-uuid-1",
			"type":       "slack",
			"name":       "Slack Alerts Renamed",
			"config":     map[string]any{"webhook_url": "https://hooks.slack.com/services/T0001/B0001/newsecret"},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-02T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	nc, err := c.UpdateNotificationChannel(context.Background(), "nc-uuid-1", client.UpdateNotificationChannelRequest{
		Name: "Slack Alerts Renamed",
		Config: client.NotificationChannelConfig{
			"webhook_url": "https://hooks.slack.com/services/T0001/B0001/newsecret",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nc.Name != "Slack Alerts Renamed" {
		t.Errorf("expected name Slack Alerts Renamed, got %s", nc.Name)
	}
	if nc.UpdatedAt != "2024-01-02T00:00:00Z" {
		t.Errorf("expected updated_at 2024-01-02T00:00:00Z, got %s", nc.UpdatedAt)
	}
}

func TestDeleteNotificationChannel_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	if err := c.DeleteNotificationChannel(context.Background(), "nc-uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteNotificationChannel_NotFound(t *testing.T) {
	srv := newTestServer(t, http.MethodDelete, "/api/v1/notification-channels/gone", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	err := c.DeleteNotificationChannel(context.Background(), "gone")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true")
	}
}

func TestListNotificationChannels_Pagination(t *testing.T) {
	call := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		call++
		var resp map[string]any
		if call == 1 {
			nextURL := srv.URL + "/api/v1/notification-channels?cursor=page2"
			resp = map[string]any{
				"data": []map[string]any{
					{"id": "nc1", "type": "slack", "name": "channel-1", "config": map[string]any{"webhook_url": "https://hooks.slack.com/1"}, "is_active": true, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
				},
				"links": map[string]any{"next": nextURL},
			}
		} else {
			resp = map[string]any{
				"data": []map[string]any{
					{"id": "nc2", "type": "email", "name": "channel-2", "config": map[string]any{"recipients": []string{"a@example.com"}}, "is_active": true, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
				},
				"links": map[string]any{"next": nil},
			}
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	channels, err := c.ListNotificationChannels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
	if channels[0].ID != "nc1" || channels[1].ID != "nc2" {
		t.Errorf("unexpected channel IDs: %v %v", channels[0].ID, channels[1].ID)
	}
}

// ---------- Audit Log Stream tests ----------

func TestCreateAuditLogStream_Webhook(t *testing.T) {
	srv := newTestServer(t, http.MethodPost, "/api/v1/audit-log-streams", 201, map[string]any{
		"data": map[string]any{
			"id":   "als-uuid-1",
			"type": "webhook",
			"config": map[string]any{
				"url":    "https://example.com/audit",
				"secret": "mysecret",
			},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	stream, err := c.CreateAuditLogStream(context.Background(), client.CreateAuditLogStreamRequest{
		Type: "webhook",
		Config: client.AuditLogStreamConfig{
			URL:    "https://example.com/audit",
			Secret: "mysecret",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.ID != "als-uuid-1" {
		t.Errorf("expected ID als-uuid-1, got %s", stream.ID)
	}
	if stream.Type != "webhook" {
		t.Errorf("expected type webhook, got %s", stream.Type)
	}
	if stream.Config.URL != "https://example.com/audit" {
		t.Errorf("expected config.url https://example.com/audit, got %s", stream.Config.URL)
	}
	if stream.Config.Secret != "mysecret" {
		t.Errorf("expected config.secret mysecret, got %s", stream.Config.Secret)
	}
	if !stream.IsActive {
		t.Error("expected is_active=true")
	}
}

func TestCreateAuditLogStream_S3(t *testing.T) {
	srv := newTestServer(t, http.MethodPost, "/api/v1/audit-log-streams", 201, map[string]any{
		"data": map[string]any{
			"id":   "als-uuid-2",
			"type": "s3",
			"config": map[string]any{
				"bucket":            "my-audit-bucket",
				"region":            "us-east-1",
				"prefix":            "lockwave/",
				"access_key_id":     "AKIAIOSFODNN7EXAMPLE",
				"secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	stream, err := c.CreateAuditLogStream(context.Background(), client.CreateAuditLogStreamRequest{
		Type: "s3",
		Config: client.AuditLogStreamConfig{
			Bucket:          "my-audit-bucket",
			Region:          "us-east-1",
			Prefix:          "lockwave/",
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.Type != "s3" {
		t.Errorf("expected type s3, got %s", stream.Type)
	}
	if stream.Config.Bucket != "my-audit-bucket" {
		t.Errorf("expected config.bucket my-audit-bucket, got %s", stream.Config.Bucket)
	}
	if stream.Config.Region != "us-east-1" {
		t.Errorf("expected config.region us-east-1, got %s", stream.Config.Region)
	}
	if stream.Config.Prefix != "lockwave/" {
		t.Errorf("expected config.prefix lockwave/, got %s", stream.Config.Prefix)
	}
	if stream.Config.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("expected config.access_key_id AKIAIOSFODNN7EXAMPLE, got %s", stream.Config.AccessKeyID)
	}
	if stream.Config.SecretAccessKey != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("expected config.secret_access_key to match, got %s", stream.Config.SecretAccessKey)
	}
}

func TestGetAuditLogStream_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/audit-log-streams/als-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":   "als-uuid-1",
			"type": "webhook",
			"config": map[string]any{
				"url": "https://example.com/audit",
			},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	stream, err := c.GetAuditLogStream(context.Background(), "als-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.ID != "als-uuid-1" {
		t.Errorf("expected ID als-uuid-1, got %s", stream.ID)
	}
	if stream.Config.URL != "https://example.com/audit" {
		t.Errorf("expected config.url https://example.com/audit, got %s", stream.Config.URL)
	}
}

func TestGetAuditLogStream_NotFound(t *testing.T) {
	srv := newTestServer(t, http.MethodGet, "/api/v1/audit-log-streams/missing", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	_, err := c.GetAuditLogStream(context.Background(), "missing")
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got err: %v", err)
	}
}

func TestUpdateAuditLogStream_Success(t *testing.T) {
	srv := newTestServer(t, http.MethodPatch, "/api/v1/audit-log-streams/als-uuid-1", 200, map[string]any{
		"data": map[string]any{
			"id":   "als-uuid-1",
			"type": "webhook",
			"config": map[string]any{
				"url": "https://example.com/audit-v2",
			},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-02T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	stream, err := c.UpdateAuditLogStream(context.Background(), "als-uuid-1", client.UpdateAuditLogStreamRequest{
		Config: client.AuditLogStreamConfig{
			URL: "https://example.com/audit-v2",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.Config.URL != "https://example.com/audit-v2" {
		t.Errorf("expected config.url https://example.com/audit-v2, got %s", stream.Config.URL)
	}
	if stream.UpdatedAt != "2024-01-02T00:00:00Z" {
		t.Errorf("expected updated_at 2024-01-02T00:00:00Z, got %s", stream.UpdatedAt)
	}
}

func TestDeleteAuditLogStream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/audit-log-streams/als-uuid-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	if err := c.DeleteAuditLogStream(context.Background(), "als-uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAuditLogStream_NotFound_IsNotFound(t *testing.T) {
	srv := newTestServer(t, http.MethodDelete, "/api/v1/audit-log-streams/gone", 404, map[string]any{
		"message": "Not found.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	err := c.DeleteAuditLogStream(context.Background(), "gone")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got err: %v", err)
	}
}

func TestListAuditLogStreams_Pagination(t *testing.T) {
	call := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		call++
		var resp map[string]any
		if call == 1 {
			nextURL := srv.URL + "/api/v1/audit-log-streams?cursor=page2"
			resp = map[string]any{
				"data": []map[string]any{
					{
						"id": "als1", "type": "webhook",
						"config":     map[string]any{"url": "https://example.com/a"},
						"is_active":  true,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z",
					},
				},
				"links": map[string]any{"next": nextURL},
			}
		} else {
			resp = map[string]any{
				"data": []map[string]any{
					{
						"id": "als2", "type": "s3",
						"config":     map[string]any{"bucket": "b", "region": "us-east-1"},
						"is_active":  false,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z",
					},
				},
				"links": map[string]any{"next": nil},
			}
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := client.NewClient(srv.URL, "test-token", "team-id")
	streams, err := c.ListAuditLogStreams(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streams) != 2 {
		t.Errorf("expected 2 streams, got %d", len(streams))
	}
	if streams[0].ID != "als1" || streams[1].ID != "als2" {
		t.Errorf("unexpected stream IDs: %v %v", streams[0].ID, streams[1].ID)
	}
	if streams[1].Type != "s3" {
		t.Errorf("expected second stream type s3, got %s", streams[1].Type)
	}
}

// ---------- IsNotFound works with wrapped errors ----------

func TestIsNotFound_WrappedError(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", &client.APIError{StatusCode: 404, Message: "not found"})
	if !client.IsNotFound(wrapped) {
		t.Error("expected IsNotFound=true for wrapped 404 APIError")
	}
}

func TestIsNotFound_NonAPIError(t *testing.T) {
	err := fmt.Errorf("generic error")
	if client.IsNotFound(err) {
		t.Error("expected IsNotFound=false for non-APIError")
	}
}
