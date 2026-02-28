---
page_title: "lockwave_host Data Source - Lockwave"
subcategory: ""
description: |-
  Fetches a single Lockwave host by ID.
---

# lockwave_host (Data Source)

Fetches a single Lockwave host by ID.

## Example Usage

```terraform
data "lockwave_host" "web01" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}

output "host_status" {
  value = data.lockwave_host.web01.status
}
```

## Schema

### Required

- `id` (String) - UUID of the host.

### Read-Only

- `display_name` (String) - Human-readable display name for the host.
- `hostname` (String) - DNS name or IP address of the host.
- `os` (String) - Operating system.
- `arch` (String) - CPU architecture.
- `status` (String) - Current sync status reported by the daemon.
- `daemon_version` (String) - Version of the Lockwave daemon running on this host.
- `last_seen_at` (String) - ISO 8601 timestamp of the last daemon sync, or null.
- `created_at` (String) - ISO 8601 creation timestamp.
- `host_users` (List) - OS users registered on this host.
