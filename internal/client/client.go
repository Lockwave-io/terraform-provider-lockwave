// Package client provides an HTTP client for the Lockwave API v1.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultBaseURL    = "https://lockwave.io"
	defaultAPIVersion = "v1"
	defaultTimeout    = 30 * time.Second
)

// Client is the Lockwave API HTTP client.
type Client struct {
	baseURL    string
	apiToken   string
	teamID     string
	httpClient *http.Client
}

// NewClient constructs a new Lockwave API client. If baseURL is empty it defaults
// to "https://lockwave.io".
func NewClient(baseURL, apiToken, teamID string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		baseURL:  baseURL,
		apiToken: apiToken,
		teamID:   teamID,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// APIError represents an error returned by the Lockwave API.
type APIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("lockwave API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("lockwave API error (status %d): %s", e.StatusCode, e.Body)
}

// errorResponse is the shape of error payloads from Laravel validation / general errors.
type errorResponse struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}

// Host represents a Lockwave host.
type Host struct {
	ID            string      `json:"id"`
	DisplayName   string      `json:"display_name"`
	Hostname      string      `json:"hostname"`
	OS            string      `json:"os"`
	Arch          string      `json:"arch"`
	Status        string      `json:"status"`
	DaemonVersion string      `json:"daemon_version"`
	LastSeenAt    *string     `json:"last_seen_at"`
	HostUsers     []HostUser  `json:"host_users"`
	Credential    string      `json:"credential,omitempty"`
	CreatedAt     string      `json:"created_at"`
}

// HostUser represents an OS user on a Lockwave host.
type HostUser struct {
	ID                   string  `json:"id"`
	OsUser               string  `json:"os_user"`
	AuthorizedKeysPath   *string `json:"authorized_keys_path"`
	AssignmentCount      *int    `json:"assignment_count,omitempty"`
	CreatedAt            string  `json:"created_at"`
}

// SshKey represents a Lockwave SSH key.
type SshKey struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	PublicKey          string  `json:"public_key,omitempty"`
	FingerprintSHA256  string  `json:"fingerprint_sha256"`
	KeyType            string  `json:"key_type"`
	KeyBits            *int    `json:"key_bits"`
	Comment            *string `json:"comment"`
	BlockedUntil       *string `json:"blocked_until"`
	BlockedIndefinite  bool    `json:"blocked_indefinite"`
	PrivateKey         string  `json:"private_key,omitempty"`
	CreatedAt          string  `json:"created_at"`
}

// Assignment represents a Lockwave SSH key-to-host-user assignment.
type Assignment struct {
	ID        string   `json:"id"`
	SshKey    *SshKey  `json:"ssh_key,omitempty"`
	HostUser  *HostUser `json:"host_user,omitempty"`
	SshKeyID  string   `json:"ssh_key_id,omitempty"`
	HostUserID string  `json:"host_user_id,omitempty"`
	ExpiresAt *string  `json:"expires_at"`
	CreatedAt string   `json:"created_at"`
}

// WebhookEndpoint represents a Lockwave webhook endpoint.
type WebhookEndpoint struct {
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	Description  *string  `json:"description"`
	Events       []string `json:"events"`
	IsActive     bool     `json:"is_active"`
	FailureCount int      `json:"failure_count"`
	DisabledAt   *string  `json:"disabled_at"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// Team represents a Lockwave team.
type Team struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	PersonalTeam bool    `json:"personal_team"`
	CreatedAt    string  `json:"created_at"`
}

// dataWrapper is the standard Laravel API resource single-item envelope.
type dataWrapper[T any] struct {
	Data T `json:"data"`
}

// listWrapper is the standard Laravel cursor-paginated collection envelope.
type listWrapper[T any] struct {
	Data []T `json:"data"`
	Links struct {
		Next *string `json:"next"`
	} `json:"links"`
}

// apiURL assembles a full URL for an API path.
func (c *Client) apiURL(path string) string {
	return fmt.Sprintf("%s/api/%s/%s", c.baseURL, defaultAPIVersion, path)
}

// do performs an HTTP request with the shared auth headers and decodes the response.
func (c *Client) do(ctx context.Context, method, rawURL string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Team-ID", c.teamID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
		var errResp errorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil {
			msg := errResp.Message
			for field, msgs := range errResp.Errors {
				for _, m := range msgs {
					msg += fmt.Sprintf("; %s: %s", field, m)
				}
			}
			apiErr.Message = msg
		}
		return nil, resp.StatusCode, apiErr
	}

	return respBody, resp.StatusCode, nil
}

// ---------- Hosts ----------

// CreateHostRequest is the payload for POST /api/v1/hosts.
type CreateHostRequest struct {
	DisplayName string `json:"display_name"`
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Arch        string `json:"arch,omitempty"`
}

// UpdateHostRequest is the payload for PATCH /api/v1/hosts/{id}.
type UpdateHostRequest struct {
	DisplayName string `json:"display_name,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
	OS          string `json:"os,omitempty"`
	Arch        string `json:"arch,omitempty"`
}

// CreateHost creates a new host and returns it along with a one-time credential.
func (c *Client) CreateHost(ctx context.Context, req CreateHostRequest) (*Host, error) {
	body, _, err := c.do(ctx, http.MethodPost, c.apiURL("hosts"), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[Host]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode host response: %w", err)
	}
	return &wrapped.Data, nil
}

// GetHost retrieves a host by ID.
func (c *Client) GetHost(ctx context.Context, id string) (*Host, error) {
	body, _, err := c.do(ctx, http.MethodGet, c.apiURL("hosts/"+url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[Host]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode host response: %w", err)
	}
	return &wrapped.Data, nil
}

// UpdateHost updates a host by ID.
func (c *Client) UpdateHost(ctx context.Context, id string, req UpdateHostRequest) (*Host, error) {
	body, _, err := c.do(ctx, http.MethodPatch, c.apiURL("hosts/"+url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[Host]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode host response: %w", err)
	}
	return &wrapped.Data, nil
}

// DeleteHost deletes a host by ID.
func (c *Client) DeleteHost(ctx context.Context, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, c.apiURL("hosts/"+url.PathEscape(id)), nil)
	return err
}

// ListHosts returns all hosts (follows cursor pagination).
func (c *Client) ListHosts(ctx context.Context) ([]Host, error) {
	return fetchAll[Host](ctx, c, c.apiURL("hosts"))
}

// ---------- Host Users ----------

// CreateHostUserRequest is the payload for POST /api/v1/hosts/{host_id}/users.
type CreateHostUserRequest struct {
	OsUser             string  `json:"os_user"`
	AuthorizedKeysPath *string `json:"authorized_keys_path,omitempty"`
}

// UpdateHostUserRequest is the payload for PATCH /api/v1/hosts/{host_id}/users/{id}.
type UpdateHostUserRequest struct {
	OsUser             string  `json:"os_user,omitempty"`
	AuthorizedKeysPath *string `json:"authorized_keys_path,omitempty"`
}

// CreateHostUser creates an OS user record on a host.
func (c *Client) CreateHostUser(ctx context.Context, hostID string, req CreateHostUserRequest) (*HostUser, error) {
	path := fmt.Sprintf("hosts/%s/users", url.PathEscape(hostID))
	body, _, err := c.do(ctx, http.MethodPost, c.apiURL(path), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[HostUser]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode host user response: %w", err)
	}
	return &wrapped.Data, nil
}

// UpdateHostUser updates an OS user record on a host.
func (c *Client) UpdateHostUser(ctx context.Context, hostID, userID string, req UpdateHostUserRequest) (*HostUser, error) {
	path := fmt.Sprintf("hosts/%s/users/%s", url.PathEscape(hostID), url.PathEscape(userID))
	body, _, err := c.do(ctx, http.MethodPatch, c.apiURL(path), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[HostUser]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode host user response: %w", err)
	}
	return &wrapped.Data, nil
}

// DeleteHostUser deletes an OS user record from a host.
func (c *Client) DeleteHostUser(ctx context.Context, hostID, userID string) error {
	path := fmt.Sprintf("hosts/%s/users/%s", url.PathEscape(hostID), url.PathEscape(userID))
	_, _, err := c.do(ctx, http.MethodDelete, c.apiURL(path), nil)
	return err
}

// GetHostUser retrieves a host user by scanning the host's user list.
// The Lockwave API does not expose a standalone GET /host-users/{id} endpoint;
// host users are nested under their parent host.
func (c *Client) GetHostUser(ctx context.Context, hostID, userID string) (*HostUser, error) {
	path := fmt.Sprintf("hosts/%s/users", url.PathEscape(hostID))
	users, err := fetchAll[HostUser](ctx, c, c.apiURL(path))
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].ID == userID {
			return &users[i], nil
		}
	}
	return nil, &APIError{StatusCode: 404, Message: fmt.Sprintf("host user %s not found on host %s", userID, hostID)}
}

// ---------- SSH Keys ----------

// CreateSshKeyRequest is the payload for POST /api/v1/ssh-keys.
type CreateSshKeyRequest struct {
	Name      string  `json:"name"`
	Mode      string  `json:"mode"`
	PublicKey *string `json:"public_key,omitempty"`
	KeyType   *string `json:"key_type,omitempty"`
	KeyBits   *int    `json:"key_bits,omitempty"`
}

// UpdateSshKeyRequest is the payload for PATCH /api/v1/ssh-keys/{id}.
type UpdateSshKeyRequest struct {
	Name    string `json:"name,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// CreateSshKey creates a new SSH key.
func (c *Client) CreateSshKey(ctx context.Context, req CreateSshKeyRequest) (*SshKey, error) {
	body, _, err := c.do(ctx, http.MethodPost, c.apiURL("ssh-keys"), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[SshKey]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode ssh key response: %w", err)
	}
	return &wrapped.Data, nil
}

// GetSshKey retrieves an SSH key by ID.
func (c *Client) GetSshKey(ctx context.Context, id string) (*SshKey, error) {
	body, _, err := c.do(ctx, http.MethodGet, c.apiURL("ssh-keys/"+url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[SshKey]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode ssh key response: %w", err)
	}
	return &wrapped.Data, nil
}

// UpdateSshKey updates an SSH key by ID.
func (c *Client) UpdateSshKey(ctx context.Context, id string, req UpdateSshKeyRequest) (*SshKey, error) {
	body, _, err := c.do(ctx, http.MethodPatch, c.apiURL("ssh-keys/"+url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[SshKey]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode ssh key response: %w", err)
	}
	return &wrapped.Data, nil
}

// DeleteSshKey deletes an SSH key by ID.
func (c *Client) DeleteSshKey(ctx context.Context, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, c.apiURL("ssh-keys/"+url.PathEscape(id)), nil)
	return err
}

// ListSshKeys returns all SSH keys (follows cursor pagination).
func (c *Client) ListSshKeys(ctx context.Context) ([]SshKey, error) {
	return fetchAll[SshKey](ctx, c, c.apiURL("ssh-keys"))
}

// ---------- Assignments ----------

// CreateAssignmentRequest is the payload for POST /api/v1/assignments.
type CreateAssignmentRequest struct {
	SshKeyID   string  `json:"ssh_key_id"`
	HostUserID string  `json:"host_user_id"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
}

// CreateAssignment creates a key-to-host-user assignment.
func (c *Client) CreateAssignment(ctx context.Context, req CreateAssignmentRequest) (*Assignment, error) {
	body, _, err := c.do(ctx, http.MethodPost, c.apiURL("assignments"), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[Assignment]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode assignment response: %w", err)
	}
	return &wrapped.Data, nil
}

// GetAssignment retrieves an assignment by ID.
func (c *Client) GetAssignment(ctx context.Context, id string) (*Assignment, error) {
	body, _, err := c.do(ctx, http.MethodGet, c.apiURL("assignments/"+url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[Assignment]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode assignment response: %w", err)
	}
	return &wrapped.Data, nil
}

// DeleteAssignment deletes an assignment by ID.
func (c *Client) DeleteAssignment(ctx context.Context, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, c.apiURL("assignments/"+url.PathEscape(id)), nil)
	return err
}

// ---------- Webhook Endpoints ----------

// CreateWebhookEndpointRequest is the payload for POST /api/v1/webhook-endpoints.
type CreateWebhookEndpointRequest struct {
	URL         string   `json:"url"`
	Description *string  `json:"description,omitempty"`
	Events      []string `json:"events"`
}

// UpdateWebhookEndpointRequest is the payload for PATCH /api/v1/webhook-endpoints/{id}.
type UpdateWebhookEndpointRequest struct {
	URL         string   `json:"url,omitempty"`
	Description *string  `json:"description,omitempty"`
	Events      []string `json:"events,omitempty"`
}

// CreateWebhookEndpoint creates a new webhook endpoint.
func (c *Client) CreateWebhookEndpoint(ctx context.Context, req CreateWebhookEndpointRequest) (*WebhookEndpoint, error) {
	body, _, err := c.do(ctx, http.MethodPost, c.apiURL("webhook-endpoints"), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[WebhookEndpoint]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode webhook endpoint response: %w", err)
	}
	return &wrapped.Data, nil
}

// GetWebhookEndpoint retrieves a webhook endpoint by ID.
func (c *Client) GetWebhookEndpoint(ctx context.Context, id string) (*WebhookEndpoint, error) {
	body, _, err := c.do(ctx, http.MethodGet, c.apiURL("webhook-endpoints/"+url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[WebhookEndpoint]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode webhook endpoint response: %w", err)
	}
	return &wrapped.Data, nil
}

// UpdateWebhookEndpoint updates a webhook endpoint by ID.
func (c *Client) UpdateWebhookEndpoint(ctx context.Context, id string, req UpdateWebhookEndpointRequest) (*WebhookEndpoint, error) {
	body, _, err := c.do(ctx, http.MethodPatch, c.apiURL("webhook-endpoints/"+url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[WebhookEndpoint]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode webhook endpoint response: %w", err)
	}
	return &wrapped.Data, nil
}

// DeleteWebhookEndpoint deletes a webhook endpoint by ID.
func (c *Client) DeleteWebhookEndpoint(ctx context.Context, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, c.apiURL("webhook-endpoints/"+url.PathEscape(id)), nil)
	return err
}

// ---------- Notification Channels ----------

// NotificationChannelConfig holds the type-specific configuration for a notification channel.
// For slack channels: {"webhook_url": "https://..."}.
// For email channels: {"recipients": ["a@example.com", "b@example.com"]}.
type NotificationChannelConfig map[string]any

// NotificationChannel represents a Lockwave notification channel.
type NotificationChannel struct {
	ID        string                    `json:"id"`
	Type      string                    `json:"type"`
	Name      string                    `json:"name"`
	Config    NotificationChannelConfig `json:"config"`
	IsActive  bool                      `json:"is_active"`
	CreatedAt string                    `json:"created_at"`
	UpdatedAt string                    `json:"updated_at"`
}

// CreateNotificationChannelRequest is the payload for POST /api/v1/notification-channels.
type CreateNotificationChannelRequest struct {
	Type   string                    `json:"type"`
	Name   string                    `json:"name"`
	Config NotificationChannelConfig `json:"config"`
}

// UpdateNotificationChannelRequest is the payload for PATCH /api/v1/notification-channels/{id}.
type UpdateNotificationChannelRequest struct {
	Name   string                    `json:"name,omitempty"`
	Config NotificationChannelConfig `json:"config,omitempty"`
}

// CreateNotificationChannel creates a new notification channel.
func (c *Client) CreateNotificationChannel(ctx context.Context, req CreateNotificationChannelRequest) (*NotificationChannel, error) {
	body, _, err := c.do(ctx, http.MethodPost, c.apiURL("notification-channels"), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[NotificationChannel]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode notification channel response: %w", err)
	}
	return &wrapped.Data, nil
}

// GetNotificationChannel retrieves a notification channel by ID.
func (c *Client) GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	body, _, err := c.do(ctx, http.MethodGet, c.apiURL("notification-channels/"+url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[NotificationChannel]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode notification channel response: %w", err)
	}
	return &wrapped.Data, nil
}

// UpdateNotificationChannel updates a notification channel by ID.
func (c *Client) UpdateNotificationChannel(ctx context.Context, id string, req UpdateNotificationChannelRequest) (*NotificationChannel, error) {
	body, _, err := c.do(ctx, http.MethodPatch, c.apiURL("notification-channels/"+url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[NotificationChannel]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode notification channel response: %w", err)
	}
	return &wrapped.Data, nil
}

// DeleteNotificationChannel deletes a notification channel by ID.
func (c *Client) DeleteNotificationChannel(ctx context.Context, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, c.apiURL("notification-channels/"+url.PathEscape(id)), nil)
	return err
}

// ListNotificationChannels returns all notification channels (follows cursor pagination).
func (c *Client) ListNotificationChannels(ctx context.Context) ([]NotificationChannel, error) {
	return fetchAll[NotificationChannel](ctx, c, c.apiURL("notification-channels"))
}

// ---------- Audit Log Streams ----------

// AuditLogStreamConfig holds the type-specific configuration for an audit log stream.
// For "webhook" type: URL and optionally Secret are used.
// For "s3" type: Bucket, Region, Prefix (optional), AccessKeyID, and SecretAccessKey are used.
type AuditLogStreamConfig struct {
	// webhook fields
	URL    string `json:"url,omitempty"`
	Secret string `json:"secret,omitempty"`

	// s3 fields
	Bucket          string `json:"bucket,omitempty"`
	Region          string `json:"region,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
}

// AuditLogStream represents a Lockwave audit log stream.
type AuditLogStream struct {
	ID        string               `json:"id"`
	Type      string               `json:"type"`
	Config    AuditLogStreamConfig `json:"config"`
	IsActive  bool                 `json:"is_active"`
	CreatedAt string               `json:"created_at"`
	UpdatedAt string               `json:"updated_at"`
}

// CreateAuditLogStreamRequest is the payload for POST /api/v1/audit-log-streams.
type CreateAuditLogStreamRequest struct {
	Type   string               `json:"type"`
	Config AuditLogStreamConfig `json:"config"`
}

// UpdateAuditLogStreamRequest is the payload for PATCH /api/v1/audit-log-streams/{id}.
type UpdateAuditLogStreamRequest struct {
	Config AuditLogStreamConfig `json:"config"`
}

// CreateAuditLogStream creates a new audit log stream.
func (c *Client) CreateAuditLogStream(ctx context.Context, req CreateAuditLogStreamRequest) (*AuditLogStream, error) {
	body, _, err := c.do(ctx, http.MethodPost, c.apiURL("audit-log-streams"), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[AuditLogStream]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode audit log stream response: %w", err)
	}
	return &wrapped.Data, nil
}

// GetAuditLogStream retrieves an audit log stream by ID.
func (c *Client) GetAuditLogStream(ctx context.Context, id string) (*AuditLogStream, error) {
	body, _, err := c.do(ctx, http.MethodGet, c.apiURL("audit-log-streams/"+url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[AuditLogStream]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode audit log stream response: %w", err)
	}
	return &wrapped.Data, nil
}

// UpdateAuditLogStream updates an audit log stream config by ID.
// The stream type is immutable; only the config may be changed.
func (c *Client) UpdateAuditLogStream(ctx context.Context, id string, req UpdateAuditLogStreamRequest) (*AuditLogStream, error) {
	body, _, err := c.do(ctx, http.MethodPatch, c.apiURL("audit-log-streams/"+url.PathEscape(id)), req)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[AuditLogStream]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode audit log stream response: %w", err)
	}
	return &wrapped.Data, nil
}

// DeleteAuditLogStream deletes an audit log stream by ID.
func (c *Client) DeleteAuditLogStream(ctx context.Context, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, c.apiURL("audit-log-streams/"+url.PathEscape(id)), nil)
	return err
}

// ListAuditLogStreams returns all audit log streams (follows cursor pagination).
func (c *Client) ListAuditLogStreams(ctx context.Context) ([]AuditLogStream, error) {
	return fetchAll[AuditLogStream](ctx, c, c.apiURL("audit-log-streams"))
}

// ---------- Teams ----------

// GetCurrentTeam retrieves the current team for the authenticated user.
func (c *Client) GetCurrentTeam(ctx context.Context) (*Team, error) {
	body, _, err := c.do(ctx, http.MethodGet, c.apiURL("teams/"+url.PathEscape(c.teamID)), nil)
	if err != nil {
		return nil, err
	}
	var wrapped dataWrapper[Team]
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode team response: %w", err)
	}
	return &wrapped.Data, nil
}

// ---------- Pagination helper ----------

// maxPages is a hard cap on the number of pages fetched to prevent infinite loops
// caused by a misbehaving server that always returns a next link.
const maxPages = 10_000

// fetchAll follows cursor pagination and accumulates all items.
func fetchAll[T any](ctx context.Context, c *Client, initialURL string) ([]T, error) {
	var all []T
	nextURL := initialURL

	for i := 0; nextURL != "" && i < maxPages; i++ {
		body, _, err := c.do(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, err
		}
		var page listWrapper[T]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode paginated response: %w", err)
		}
		all = append(all, page.Data...)
		if page.Links.Next == nil || *page.Links.Next == "" {
			break
		}
		nextURL = *page.Links.Next
	}

	return all, nil
}

// IsNotFound returns true when the error represents an HTTP 404.
// It uses errors.As so it works correctly with wrapped errors.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}
