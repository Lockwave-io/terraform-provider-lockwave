---
page_title: "lockwave_ssh_keys Data Source - Lockwave"
subcategory: ""
description: |-
  Fetches all Lockwave SSH keys for the current team.
---

# lockwave_ssh_keys (Data Source)

Fetches all Lockwave SSH keys for the current team.

## Example Usage

```terraform
data "lockwave_ssh_keys" "all" {}

output "key_count" {
  value = length(data.lockwave_ssh_keys.all.keys)
}

output "key_names" {
  value = [for k in data.lockwave_ssh_keys.all.keys : k.name]
}
```

## Schema

### Read-Only

- `keys` (List of Object) - List of SSH keys. Each element contains:
  - `id` (String) - UUID of the SSH key.
  - `name` (String) - Human-readable name.
  - `public_key` (String) - OpenSSH public key.
  - `key_type` (String) - Key algorithm.
  - `fingerprint_sha256` (String) - SHA-256 fingerprint.
  - `comment` (String) - Comment.
  - `blocked_until` (String) - Block expiry timestamp.
  - `blocked_indefinite` (Boolean) - Whether blocked indefinitely.
  - `created_at` (String) - Creation timestamp.
