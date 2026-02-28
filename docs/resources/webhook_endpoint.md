---
page_title: "lockwave_webhook_endpoint Resource - Lockwave"
subcategory: ""
description: |-
  Manages a Lockwave webhook endpoint that receives event notifications.
---

# lockwave_webhook_endpoint (Resource)

Manages a Lockwave webhook endpoint that receives event notifications.

## Example Usage

```terraform
resource "lockwave_webhook_endpoint" "ops_alerts" {
  url         = "https://hooks.example.com/lockwave"
  description = "Operations alerts channel"
  events = [
    "host.synced",
    "ssh_key.created",
    "ssh_key.deleted",
    "assignment.created",
    "assignment.deleted",
  ]
}
```

## Schema

### Required

- `url` (String) - HTTPS URL to deliver webhook payloads to.
- `events` (List of String) - List of event types to subscribe to.

Supported event types: `host.synced`, `ssh_key.created`, `ssh_key.deleted`, `assignment.created`, `assignment.deleted`.

### Optional

- `description` (String) - Human-readable description for the webhook endpoint.

### Read-Only

- `id` (String) - UUID of the webhook endpoint.
- `is_active` (Boolean) - Whether the webhook endpoint is currently active.
- `failure_count` (Number) - Cumulative delivery failure count.
- `disabled_at` (String) - ISO 8601 timestamp when the endpoint was automatically disabled due to failures, or null.
- `created_at` (String) - ISO 8601 creation timestamp.
- `updated_at` (String) - ISO 8601 timestamp of the last update.

## Import

```shell
terraform import lockwave_webhook_endpoint.ops_alerts <endpoint-uuid>
```
