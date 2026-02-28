---
page_title: "lockwave_hosts Data Source - Lockwave"
subcategory: ""
description: |-
  Fetches all Lockwave hosts for the current team.
---

# lockwave_hosts (Data Source)

Fetches all Lockwave hosts for the current team.

## Example Usage

```terraform
data "lockwave_hosts" "all" {}

output "host_count" {
  value = length(data.lockwave_hosts.all.hosts)
}

output "host_names" {
  value = [for h in data.lockwave_hosts.all.hosts : h.display_name]
}
```

## Schema

### Read-Only

- `hosts` (List of Object) - List of hosts. Each element contains:
  - `id` (String) - UUID of the host.
  - `display_name` (String) - Human-readable display name.
  - `hostname` (String) - DNS name or IP address.
  - `os` (String) - Operating system.
  - `arch` (String) - CPU architecture.
  - `status` (String) - Current sync status.
  - `daemon_version` (String) - Daemon version.
  - `last_seen_at` (String) - Last sync timestamp.
  - `created_at` (String) - Creation timestamp.
  - `host_users` (List) - OS users on this host.
