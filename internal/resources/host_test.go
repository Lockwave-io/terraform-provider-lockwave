package resources_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lockwave-io/terraform-provider-lockwave/internal/client"
)

// mockServer creates a simple httptest.Server for a single endpoint.
func mockServer(t *testing.T, method, path string, status int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Error(err)
			}
		}
	}))
}

func TestHostResource_ClientRoundTrip(t *testing.T) {
	// Verify that the client's Create/Get/Update/Delete path works end-to-end
	// with mock responses matching the shapes the resource code expects.

	hostPayload := map[string]any{
		"data": map[string]any{
			"id":            "host-uuid",
			"display_name":  "web-01",
			"hostname":      "web-01.example.com",
			"os":            "linux",
			"arch":          "x86_64",
			"status":        "pending",
			"daemon_version": "",
			"credential":    "cred-abc",
			"created_at":    "2024-01-01T00:00:00Z",
			"host_users":    []any{},
		},
	}

	t.Run("create", func(t *testing.T) {
		srv := mockServer(t, http.MethodPost, "/api/v1/hosts", 201, hostPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		h, err := c.CreateHost(context.Background(), client.CreateHostRequest{
			DisplayName: "web-01",
			Hostname:    "web-01.example.com",
			OS:          "linux",
			Arch:        "x86_64",
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if h.ID != "host-uuid" {
			t.Errorf("id mismatch: %s", h.ID)
		}
		if h.Credential != "cred-abc" {
			t.Errorf("credential mismatch: %s", h.Credential)
		}
	})

	t.Run("read", func(t *testing.T) {
		srv := mockServer(t, http.MethodGet, "/api/v1/hosts/host-uuid", 200, hostPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		h, err := c.GetHost(context.Background(), "host-uuid")
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if h.OS != "linux" {
			t.Errorf("os mismatch: %s", h.OS)
		}
	})

	t.Run("update", func(t *testing.T) {
		updated := map[string]any{
			"data": map[string]any{
				"id":           "host-uuid",
				"display_name": "web-01-renamed",
				"hostname":     "web-01.example.com",
				"os":           "linux",
				"arch":         "x86_64",
				"status":       "synced",
				"created_at":   "2024-01-01T00:00:00Z",
			},
		}
		srv := mockServer(t, http.MethodPatch, "/api/v1/hosts/host-uuid", 200, updated)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		h, err := c.UpdateHost(context.Background(), "host-uuid", client.UpdateHostRequest{DisplayName: "web-01-renamed"})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if h.DisplayName != "web-01-renamed" {
			t.Errorf("display_name mismatch: %s", h.DisplayName)
		}
	})

	t.Run("delete", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		if err := c.DeleteHost(context.Background(), "host-uuid"); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})
}

func TestHostUserResource_ClientRoundTrip(t *testing.T) {
	path := "/Users/home/ubuntu/.ssh/authorized_keys"
	huPayload := map[string]any{
		"data": map[string]any{
			"id":                   "hu-uuid",
			"os_user":              "ubuntu",
			"authorized_keys_path": path,
			"created_at":           "2024-01-01T00:00:00Z",
		},
	}

	t.Run("create", func(t *testing.T) {
		srv := mockServer(t, http.MethodPost, "/api/v1/hosts/h1/users", 201, huPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		hu, err := c.CreateHostUser(context.Background(), "h1", client.CreateHostUserRequest{
			OsUser:             "ubuntu",
			AuthorizedKeysPath: &path,
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if hu.OsUser != "ubuntu" {
			t.Errorf("os_user mismatch: %s", hu.OsUser)
		}
		if hu.AuthorizedKeysPath == nil || *hu.AuthorizedKeysPath != path {
			t.Errorf("authorized_keys_path mismatch")
		}
	})

	t.Run("update", func(t *testing.T) {
		newPath := "/home/ubuntu/.ssh/authorized_keys"
		updated := map[string]any{
			"data": map[string]any{
				"id":                   "hu-uuid",
				"os_user":              "ubuntu",
				"authorized_keys_path": newPath,
				"created_at":           "2024-01-01T00:00:00Z",
			},
		}
		srv := mockServer(t, http.MethodPatch, "/api/v1/hosts/h1/users/hu-uuid", 200, updated)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		hu, err := c.UpdateHostUser(context.Background(), "h1", "hu-uuid", client.UpdateHostUserRequest{
			OsUser:             "ubuntu",
			AuthorizedKeysPath: &newPath,
		})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if hu.AuthorizedKeysPath == nil || *hu.AuthorizedKeysPath != newPath {
			t.Errorf("authorized_keys_path mismatch after update")
		}
	})
}

func TestSshKeyResource_ClientRoundTrip(t *testing.T) {
	keyPayload := map[string]any{
		"data": map[string]any{
			"id":                 "key-uuid",
			"name":               "deploy",
			"fingerprint_sha256": "SHA256:xyz",
			"key_type":           "ed25519",
			"blocked_indefinite": false,
			"private_key":        "-----BEGIN OPENSSH PRIVATE KEY-----",
			"created_at":         "2024-01-01T00:00:00Z",
		},
	}

	t.Run("create_generate", func(t *testing.T) {
		srv := mockServer(t, http.MethodPost, "/api/v1/ssh-keys", 201, keyPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		keyType := "ed25519"
		k, err := c.CreateSshKey(context.Background(), client.CreateSshKeyRequest{
			Name:    "deploy",
			Mode:    "generate",
			KeyType: &keyType,
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if k.PrivateKey == "" {
			t.Error("expected private key on create")
		}
	})

	t.Run("update_name", func(t *testing.T) {
		updated := map[string]any{
			"data": map[string]any{
				"id":                 "key-uuid",
				"name":               "deploy-renamed",
				"fingerprint_sha256": "SHA256:xyz",
				"key_type":           "ed25519",
				"blocked_indefinite": false,
				"created_at":         "2024-01-01T00:00:00Z",
			},
		}
		srv := mockServer(t, http.MethodPatch, "/api/v1/ssh-keys/key-uuid", 200, updated)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		k, err := c.UpdateSshKey(context.Background(), "key-uuid", client.UpdateSshKeyRequest{Name: "deploy-renamed"})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if k.Name != "deploy-renamed" {
			t.Errorf("name mismatch: %s", k.Name)
		}
	})
}

func TestWebhookEndpointResource_ClientRoundTrip(t *testing.T) {
	desc := "test hook"
	whPayload := map[string]any{
		"data": map[string]any{
			"id":            "wh-uuid",
			"url":           "https://example.com/hook",
			"description":   desc,
			"events":        []string{"host.synced"},
			"is_active":     true,
			"failure_count": 0,
			"created_at":    "2024-01-01T00:00:00Z",
			"updated_at":    "2024-01-01T00:00:00Z",
		},
	}

	t.Run("create", func(t *testing.T) {
		srv := mockServer(t, http.MethodPost, "/api/v1/webhook-endpoints", 201, whPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		wh, err := c.CreateWebhookEndpoint(context.Background(), client.CreateWebhookEndpointRequest{
			URL:         "https://example.com/hook",
			Description: &desc,
			Events:      []string{"host.synced"},
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if wh.URL != "https://example.com/hook" {
			t.Errorf("url mismatch: %s", wh.URL)
		}
		if len(wh.Events) != 1 || wh.Events[0] != "host.synced" {
			t.Errorf("events mismatch: %v", wh.Events)
		}
	})
}

func TestNotificationChannelResource_ClientRoundTrip(t *testing.T) {
	slackPayload := map[string]any{
		"data": map[string]any{
			"id":         "nc-uuid",
			"type":       "slack",
			"name":       "Ops Slack",
			"config":     map[string]any{"webhook_url": "https://hooks.slack.com/services/T/B/secret"},
			"is_active":  true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		},
	}

	t.Run("create_slack", func(t *testing.T) {
		srv := mockServer(t, http.MethodPost, "/api/v1/notification-channels", 201, slackPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		nc, err := c.CreateNotificationChannel(context.Background(), client.CreateNotificationChannelRequest{
			Type:   "slack",
			Name:   "Ops Slack",
			Config: client.NotificationChannelConfig{"webhook_url": "https://hooks.slack.com/services/T/B/secret"},
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if nc.ID != "nc-uuid" {
			t.Errorf("id mismatch: %s", nc.ID)
		}
		if nc.Type != "slack" {
			t.Errorf("type mismatch: %s", nc.Type)
		}
		if nc.Config["webhook_url"] != "https://hooks.slack.com/services/T/B/secret" {
			t.Errorf("webhook_url mismatch: %v", nc.Config["webhook_url"])
		}
	})

	t.Run("read", func(t *testing.T) {
		srv := mockServer(t, http.MethodGet, "/api/v1/notification-channels/nc-uuid", 200, slackPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		nc, err := c.GetNotificationChannel(context.Background(), "nc-uuid")
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if nc.Name != "Ops Slack" {
			t.Errorf("name mismatch: %s", nc.Name)
		}
		if !nc.IsActive {
			t.Error("expected is_active=true")
		}
	})

	t.Run("update", func(t *testing.T) {
		updated := map[string]any{
			"data": map[string]any{
				"id":         "nc-uuid",
				"type":       "slack",
				"name":       "Ops Slack Renamed",
				"config":     map[string]any{"webhook_url": "https://hooks.slack.com/services/T/B/new"},
				"is_active":  true,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z",
			},
		}
		srv := mockServer(t, http.MethodPatch, "/api/v1/notification-channels/nc-uuid", 200, updated)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		nc, err := c.UpdateNotificationChannel(context.Background(), "nc-uuid", client.UpdateNotificationChannelRequest{
			Name:   "Ops Slack Renamed",
			Config: client.NotificationChannelConfig{"webhook_url": "https://hooks.slack.com/services/T/B/new"},
		})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if nc.Name != "Ops Slack Renamed" {
			t.Errorf("name mismatch after update: %s", nc.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		if err := c.DeleteNotificationChannel(context.Background(), "nc-uuid"); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})

	t.Run("create_email", func(t *testing.T) {
		emailPayload := map[string]any{
			"data": map[string]any{
				"id":         "nc-email-uuid",
				"type":       "email",
				"name":       "Email Alerts",
				"config":     map[string]any{"recipients": []string{"ops@example.com"}},
				"is_active":  true,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z",
			},
		}
		srv := mockServer(t, http.MethodPost, "/api/v1/notification-channels", 201, emailPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		nc, err := c.CreateNotificationChannel(context.Background(), client.CreateNotificationChannelRequest{
			Type:   "email",
			Name:   "Email Alerts",
			Config: client.NotificationChannelConfig{"recipients": []string{"ops@example.com"}},
		})
		if err != nil {
			t.Fatalf("create email: %v", err)
		}
		if nc.Type != "email" {
			t.Errorf("type mismatch: %s", nc.Type)
		}
	})

	t.Run("read_not_found", func(t *testing.T) {
		srv := mockServer(t, http.MethodGet, "/api/v1/notification-channels/gone", 404, map[string]any{
			"message": "Not found.",
		})
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		_, err := c.GetNotificationChannel(context.Background(), "gone")
		if !client.IsNotFound(err) {
			t.Errorf("expected IsNotFound=true, got err: %v", err)
		}
	})
}

func TestAssignmentResource_ClientRoundTrip(t *testing.T) {
	assignPayload := map[string]any{
		"data": map[string]any{
			"id": "assign-uuid",
			"ssh_key": map[string]any{
				"id":   "key-uuid",
				"name": "deploy",
			},
			"host_user": map[string]any{
				"id":      "hu-uuid",
				"os_user": "ubuntu",
			},
			"created_at": "2024-01-01T00:00:00Z",
		},
	}

	t.Run("create", func(t *testing.T) {
		srv := mockServer(t, http.MethodPost, "/api/v1/assignments", 201, assignPayload)
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		a, err := c.CreateAssignment(context.Background(), client.CreateAssignmentRequest{
			SshKeyID:   "key-uuid",
			HostUserID: "hu-uuid",
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if a.ID != "assign-uuid" {
			t.Errorf("id mismatch: %s", a.ID)
		}
	})

	t.Run("delete", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()
		c := client.NewClient(srv.URL, "token", "team")
		if err := c.DeleteAssignment(context.Background(), "assign-uuid"); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})
}
