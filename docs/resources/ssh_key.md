---
page_title: "lockwave_ssh_key Resource - Lockwave"
subcategory: ""
description: |-
  Manages a Lockwave SSH key. Keys can be server-generated or supplied by the caller.
---

# lockwave_ssh_key (Resource)

Manages a Lockwave SSH key. Keys can be server-generated (`mode = "generate"`) or supplied by the caller (`mode = "import"`).

## Example Usage

### Server-Generated Key

```terraform
resource "lockwave_ssh_key" "deploy" {
  name     = "deploy-key"
  mode     = "generate"
  key_type = "ed25519"
}

resource "local_sensitive_file" "deploy_private_key" {
  content         = lockwave_ssh_key.deploy.private_key
  filename        = "${path.module}/deploy_key"
  file_permission = "0600"
}
```

### Imported Key

```terraform
resource "lockwave_ssh_key" "existing" {
  name       = "existing-key"
  mode       = "import"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG..."
}
```

## Schema

### Required

- `name` (String) - Human-readable name for the SSH key.
- `mode` (String) - Creation mode. One of: `generate`, `import`. Changing this forces a new resource.

### Optional

- `key_type` (String) - Key algorithm. One of: `ed25519`, `rsa`. Required when `mode = "generate"`. Changing this forces a new resource.
- `key_bits` (Number) - RSA key size. One of: `3072`, `4096`. Only relevant when `key_type = "rsa"`.
- `public_key` (String) - OpenSSH public key. Required when `mode = "import"`. Changing this forces a new resource.
- `comment` (String) - Optional comment embedded in the public key.

### Read-Only

- `id` (String) - UUID of the SSH key.
- `fingerprint_sha256` (String) - SHA-256 fingerprint of the public key.
- `blocked_until` (String) - ISO 8601 timestamp until which the key is blocked, or null.
- `blocked_indefinite` (Boolean) - Whether the key is blocked indefinitely.
- `private_key` (String, Sensitive) - Private key returned on creation when `mode = "generate"`. Store this securely; it is never returned again.
- `created_at` (String) - ISO 8601 creation timestamp.

## Import

```shell
terraform import lockwave_ssh_key.deploy <key-uuid>
```

~> **Note:** Imported keys always use `mode = "import"` in state. The `private_key` attribute will be empty after import.
