package datasources_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fwartner/terraform-provider-lockwave/internal/client"
)

// newDSTestServer creates an httptest.Server that responds to requests with the
// given status code and JSON body regardless of path.
func newDSTestServer(t *testing.T, status int, body any) *httptest.Server {
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

// ---------- Host data source client round-trip tests ----------

func TestHostDataSource_ClientRead(t *testing.T) {
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":             "host-uuid",
			"display_name":   "web-01",
			"hostname":       "web-01.example.com",
			"os":             "linux",
			"arch":           "x86_64",
			"status":         "synced",
			"daemon_version": "1.2.0",
			"last_seen_at":   "2024-06-01T12:00:00Z",
			"created_at":     "2024-01-01T00:00:00Z",
			"host_users":     []any{},
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	host, err := c.GetHost(context.Background(), "host-uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host.ID != "host-uuid" {
		t.Errorf("expected id host-uuid, got %s", host.ID)
	}
	if host.Status != "synced" {
		t.Errorf("expected status synced, got %s", host.Status)
	}
	if host.DaemonVersion != "1.2.0" {
		t.Errorf("expected daemon_version 1.2.0, got %s", host.DaemonVersion)
	}
	if host.LastSeenAt == nil || *host.LastSeenAt != "2024-06-01T12:00:00Z" {
		t.Errorf("unexpected last_seen_at: %v", host.LastSeenAt)
	}
}

func TestHostDataSource_ClientRead_WithHostUsers(t *testing.T) {
	akp := "/home/ubuntu/.ssh/authorized_keys"
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":           "host-uuid",
			"display_name": "web-01",
			"hostname":     "web-01.example.com",
			"os":           "linux",
			"arch":         "x86_64",
			"status":       "synced",
			"created_at":   "2024-01-01T00:00:00Z",
			"host_users": []any{
				map[string]any{
					"id":                   "hu-uuid",
					"os_user":              "ubuntu",
					"authorized_keys_path": akp,
					"created_at":           "2024-01-01T00:00:00Z",
				},
			},
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	host, err := c.GetHost(context.Background(), "host-uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(host.HostUsers) != 1 {
		t.Fatalf("expected 1 host user, got %d", len(host.HostUsers))
	}
	if host.HostUsers[0].OsUser != "ubuntu" {
		t.Errorf("expected os_user ubuntu, got %s", host.HostUsers[0].OsUser)
	}
	if host.HostUsers[0].AuthorizedKeysPath == nil || *host.HostUsers[0].AuthorizedKeysPath != akp {
		t.Errorf("unexpected authorized_keys_path: %v", host.HostUsers[0].AuthorizedKeysPath)
	}
}

// ---------- Hosts data source client round-trip tests ----------

func TestHostsDataSource_ClientListHosts_Empty(t *testing.T) {
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data":  []any{},
		"links": map[string]any{"next": nil},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	hosts, err := c.ListHosts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestHostsDataSource_ClientListHosts_Multiple(t *testing.T) {
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data": []any{
			map[string]any{"id": "h1", "display_name": "host-1", "hostname": "h1.local", "os": "linux", "arch": "x86_64", "status": "synced", "created_at": "2024-01-01T00:00:00Z"},
			map[string]any{"id": "h2", "display_name": "host-2", "hostname": "h2.local", "os": "darwin", "arch": "aarch64", "status": "pending", "created_at": "2024-01-02T00:00:00Z"},
		},
		"links": map[string]any{"next": nil},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	hosts, err := c.ListHosts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(hosts))
	}
}

// ---------- SSH Key data source client round-trip tests ----------

func TestSshKeyDataSource_ClientRead(t *testing.T) {
	keyBits := 4096
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":                 "key-uuid",
			"name":               "deploy",
			"fingerprint_sha256": "SHA256:xyz",
			"key_type":           "rsa",
			"key_bits":           keyBits,
			"comment":            "deploy@ci",
			"blocked_indefinite": false,
			"created_at":         "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	key, err := c.GetSshKey(context.Background(), "key-uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.ID != "key-uuid" {
		t.Errorf("expected id key-uuid, got %s", key.ID)
	}
	if key.KeyType != "rsa" {
		t.Errorf("expected key_type rsa, got %s", key.KeyType)
	}
	if key.KeyBits == nil || *key.KeyBits != keyBits {
		t.Errorf("expected key_bits %d, got %v", keyBits, key.KeyBits)
	}
	if key.Comment == nil || *key.Comment != "deploy@ci" {
		t.Errorf("expected comment deploy@ci, got %v", key.Comment)
	}
}

func TestSshKeyDataSource_ClientRead_NullOptionalFields(t *testing.T) {
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":                 "key-uuid",
			"name":               "deploy",
			"fingerprint_sha256": "SHA256:xyz",
			"key_type":           "ed25519",
			"blocked_indefinite": false,
			"created_at":         "2024-01-01T00:00:00Z",
			// key_bits, comment, blocked_until intentionally absent (null).
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	key, err := c.GetSshKey(context.Background(), "key-uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.KeyBits != nil {
		t.Errorf("expected nil key_bits, got %v", key.KeyBits)
	}
	if key.Comment != nil {
		t.Errorf("expected nil comment, got %v", key.Comment)
	}
	if key.BlockedUntil != nil {
		t.Errorf("expected nil blocked_until, got %v", key.BlockedUntil)
	}
}

// ---------- SSH Keys data source client round-trip tests ----------

func TestSshKeysDataSource_ClientListKeys_Empty(t *testing.T) {
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data":  []any{},
		"links": map[string]any{"next": nil},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	keys, err := c.ListSshKeys(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

// ---------- Team data source client round-trip tests ----------

func TestTeamDataSource_ClientRead(t *testing.T) {
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":            "team-uuid",
			"name":          "Acme Corp",
			"personal_team": false,
			"created_at":    "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team-uuid")
	team, err := c.GetCurrentTeam(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if team.Name != "Acme Corp" {
		t.Errorf("expected name Acme Corp, got %s", team.Name)
	}
	if team.PersonalTeam {
		t.Error("expected personal_team=false")
	}
}

func TestTeamDataSource_ClientRead_PersonalTeam(t *testing.T) {
	srv := newDSTestServer(t, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":            "personal-team-uuid",
			"name":          "John's Team",
			"personal_team": true,
			"created_at":    "2024-01-01T00:00:00Z",
		},
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "personal-team-uuid")
	team, err := c.GetCurrentTeam(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !team.PersonalTeam {
		t.Error("expected personal_team=true")
	}
}

// ---------- Error handling tests ----------

func TestDataSource_ServerError(t *testing.T) {
	srv := newDSTestServer(t, http.StatusInternalServerError, map[string]any{
		"message": "Internal Server Error",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	_, err := c.GetHost(context.Background(), "any-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if client.IsNotFound(err) {
		t.Error("expected IsNotFound=false for 500 error")
	}
}

func TestDataSource_Unauthorized(t *testing.T) {
	srv := newDSTestServer(t, http.StatusUnauthorized, map[string]any{
		"message": "Unauthenticated.",
	})
	defer srv.Close()

	c := client.NewClient(srv.URL, "token", "team")
	_, err := c.GetSshKey(context.Background(), "any-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if client.IsNotFound(err) {
		t.Error("expected IsNotFound=false for 401 error")
	}
}
