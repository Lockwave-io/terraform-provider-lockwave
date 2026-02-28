---
page_title: "lockwave_host Resource - Lockwave"
subcategory: ""
description: |-
  Manages a Lockwave host. Creating a host provisions a one-time daemon credential used to bootstrap the lockwaved daemon.
---

# lockwave_host (Resource)

Manages a Lockwave host. Creating a host provisions a one-time daemon credential stored in the `credential` attribute that must be used to bootstrap the `lockwaved` daemon.

## Example Usage

```terraform
resource "lockwave_host" "web01" {
  display_name = "web-01"
  hostname     = "web-01.example.com"
  os           = "linux"
  arch         = "x86_64"
}

output "web01_credential" {
  value     = lockwave_host.web01.credential
  sensitive = true
}
```

## Schema

### Required

- `display_name` (String) - Human-readable display name for the host.
- `hostname` (String) - DNS name or IP address of the host.
- `os` (String) - Operating system. One of: `linux`, `darwin`, `freebsd`.

### Optional

- `arch` (String) - CPU architecture. One of: `x86_64`, `aarch64`, `amd64`, `arm64`. Computed if omitted.

### Read-Only

- `id` (String) - UUID of the host.
- `status` (String) - Current sync status reported by the daemon.
- `daemon_version` (String) - Version of the Lockwave daemon running on this host.
- `last_seen_at` (String) - ISO 8601 timestamp of the last daemon sync, or null.
- `credential` (String, Sensitive) - One-time daemon credential returned on creation. Store this securely; it is not available after the initial apply.
- `created_at` (String) - ISO 8601 creation timestamp.
- `host_users` (List) - OS users registered on this host (read-only nested list).

## Import

```shell
terraform import lockwave_host.web01 <host-uuid>
```

~> **Note:** The `credential` attribute cannot be recovered on import because the API does not return it after the initial creation response.
