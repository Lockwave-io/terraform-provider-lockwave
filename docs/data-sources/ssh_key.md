---
page_title: "lockwave_ssh_key Data Source - Lockwave"
subcategory: ""
description: |-
  Fetches a single Lockwave SSH key by ID.
---

# lockwave_ssh_key (Data Source)

Fetches a single Lockwave SSH key by ID.

## Example Usage

```terraform
data "lockwave_ssh_key" "deploy" {
  id = "550e8400-e29b-41d4-a716-446655440001"
}

output "key_fingerprint" {
  value = data.lockwave_ssh_key.deploy.fingerprint_sha256
}
```

## Schema

### Required

- `id` (String) - UUID of the SSH key.

### Read-Only

- `name` (String) - Human-readable name.
- `public_key` (String) - OpenSSH public key.
- `key_type` (String) - Key algorithm (`ed25519` or `rsa`).
- `fingerprint_sha256` (String) - SHA-256 fingerprint.
- `comment` (String) - Comment embedded in the public key.
- `blocked_until` (String) - ISO 8601 timestamp until which the key is blocked, or null.
- `blocked_indefinite` (Boolean) - Whether the key is blocked indefinitely.
- `created_at` (String) - ISO 8601 creation timestamp.
