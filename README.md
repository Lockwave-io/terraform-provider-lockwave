# Terraform Provider Lockwave

[![Go Version](https://img.shields.io/badge/go-1.21%2B-blue)](https://go.dev)
[![Terraform Registry](https://img.shields.io/badge/terraform-registry-purple)](https://registry.terraform.io/providers/fwartner/lockwave/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Manage SSH key lifecycle, hosts, assignments, and integrations via the [Lockwave](https://lockwave.io) SaaS API. The provider lets you declaratively provision and enforce SSH access across your Linux/Unix fleet from within Terraform.

---

## Requirements

| Component | Minimum version |
|-----------|----------------|
| Terraform | 1.0            |
| Go        | 1.21 (building from source only) |

---

## Installation

Add the provider to your `terraform` block:

```hcl
terraform {
  required_providers {
    lockwave = {
      source  = "fwartner/lockwave"
      version = "~> 0.1"
    }
  }
}
```

Run `terraform init` to download the provider from the Terraform Registry.

---

## Provider Configuration

```hcl
provider "lockwave" {
  # api_url   = "https://lockwave.io"  # optional — defaults to https://lockwave.io
  api_token = var.lockwave_api_token   # required, sensitive
  team_id   = var.lockwave_team_id     # required
}
```

### Arguments

| Argument    | Type   | Required | Description |
|-------------|--------|----------|-------------|
| `api_url`   | string | No       | Base URL of the Lockwave API. Defaults to `https://lockwave.io`. |
| `api_token` | string | Yes      | Sanctum Bearer token for authenticating with the Lockwave API. Marked sensitive. |
| `team_id`   | string | Yes      | UUID of the Lockwave team that all resources belong to. |

### Environment Variables

All three arguments can be supplied via environment variables instead of (or in addition to) the provider block:

| Argument    | Environment variable    |
|-------------|------------------------|
| `api_url`   | `LOCKWAVE_API_URL`     |
| `api_token` | `LOCKWAVE_API_TOKEN`   |
| `team_id`   | `LOCKWAVE_TEAM_ID`     |

---

## Resources

### `lockwave_host`

Manages a Lockwave host. Creating a host provisions a one-time daemon credential stored in the `credential` attribute that must be used to bootstrap the `lockwaved` daemon.

#### Arguments

| Argument       | Type   | Required | Description |
|----------------|--------|----------|-------------|
| `display_name` | string | Yes      | Human-readable display name for the host. |
| `hostname`     | string | Yes      | DNS name or IP address of the host. |
| `os`           | string | Yes      | Operating system. One of: `linux`, `darwin`, `freebsd`. |
| `arch`         | string | No       | CPU architecture. One of: `x86_64`, `aarch64`, `amd64`, `arm64`. Computed if omitted. |

#### Attributes (computed)

| Attribute       | Type   | Description |
|-----------------|--------|-------------|
| `id`            | string | UUID of the host. |
| `status`        | string | Current sync status reported by the daemon. |
| `daemon_version`| string | Version of the Lockwave daemon running on this host. |
| `last_seen_at`  | string | ISO 8601 timestamp of the last daemon sync, or null. |
| `credential`    | string | **Sensitive. One-time only.** Daemon credential returned on creation. Store this securely; it is not available after the initial apply. |
| `created_at`    | string | ISO 8601 creation timestamp. |
| `host_users`    | list   | OS users registered on this host (read-only nested list). |

#### Import

```shell
terraform import lockwave_host.web01 <host-uuid>
```

Note: The `credential` attribute cannot be recovered on import because the API does not return it after the initial creation response.

---

### `lockwave_host_user`

Manages an OS user record on a Lockwave host. These records define which `authorized_keys` files the daemon manages.

#### Arguments

| Argument               | Type   | Required | Description |
|------------------------|--------|----------|-------------|
| `host_id`              | string | Yes      | UUID of the parent host. Changing this forces a new resource. |
| `os_user`              | string | Yes      | OS username (e.g. `ubuntu`, `ec2-user`, `deploy`). |
| `authorized_keys_path` | string | No       | Absolute path to the `authorized_keys` file. Defaults to the standard location for the OS user. Computed if omitted. |

#### Attributes (computed)

| Attribute              | Type   | Description |
|------------------------|--------|-------------|
| `id`                   | string | UUID of the host user. |
| `created_at`           | string | ISO 8601 creation timestamp. |

#### Import

Import using the composite ID format `<host_id>/<user_id>`:

```shell
terraform import lockwave_host_user.ubuntu <host-uuid>/<user-uuid>
```

---

### `lockwave_ssh_key`

Manages a Lockwave SSH key. Keys can be server-generated (`mode = "generate"`) or supplied by the caller (`mode = "import"`).

#### Arguments

| Argument     | Type   | Required | Description |
|--------------|--------|----------|-------------|
| `name`       | string | Yes      | Human-readable name for the SSH key. |
| `mode`       | string | Yes      | Creation mode. One of: `generate`, `import`. Changing this forces a new resource. |
| `key_type`   | string | No       | Key algorithm. One of: `ed25519`, `rsa`. Required when `mode = "generate"`. Changing this forces a new resource. |
| `key_bits`   | number | No       | RSA key size. One of: `3072`, `4096`. Only relevant when `key_type = "rsa"`. |
| `public_key` | string | No       | OpenSSH public key. Required when `mode = "import"`. Changing this forces a new resource. |
| `comment`    | string | No       | Optional comment embedded in the public key. |

#### Attributes (computed)

| Attribute           | Type   | Description |
|---------------------|--------|-------------|
| `id`                | string | UUID of the SSH key. |
| `fingerprint_sha256`| string | SHA-256 fingerprint of the public key. |
| `blocked_until`     | string | ISO 8601 timestamp until which the key is blocked, or null. |
| `blocked_indefinite`| bool   | Whether the key is blocked indefinitely. |
| `private_key`       | string | **Sensitive. One-time only.** Private key returned on creation when `mode = "generate"`. Store this securely; it is never returned again. |
| `created_at`        | string | ISO 8601 creation timestamp. |

#### Import

```shell
terraform import lockwave_ssh_key.deploy <key-uuid>
```

Note: Imported keys always use `mode = "import"` in state. The `private_key` attribute will be empty after import.

---

### `lockwave_assignment`

Manages a Lockwave assignment that grants an SSH key access to an OS user on a host. All input fields are ForceNew — any change recreates the assignment.

#### Arguments

| Argument       | Type   | Required | Description |
|----------------|--------|----------|-------------|
| `ssh_key_id`   | string | Yes      | UUID of the SSH key to assign. Changing this forces a new resource. |
| `host_user_id` | string | Yes      | UUID of the host user to assign the key to. Changing this forces a new resource. |
| `expires_at`   | string | No       | Optional ISO 8601 expiry timestamp for the assignment. Computed if omitted. |

#### Attributes (computed)

| Attribute   | Type   | Description |
|-------------|--------|-------------|
| `id`        | string | UUID of the assignment. |
| `created_at`| string | ISO 8601 creation timestamp. |

#### Import

```shell
terraform import lockwave_assignment.deploy_to_web01 <assignment-uuid>
```

---

### `lockwave_webhook_endpoint`

Manages a Lockwave webhook endpoint that receives event notifications.

#### Arguments

| Argument      | Type        | Required | Description |
|---------------|-------------|----------|-------------|
| `url`         | string      | Yes      | HTTPS URL to deliver webhook payloads to. |
| `events`      | list(string)| Yes      | List of event types to subscribe to. |
| `description` | string      | No       | Human-readable description for the webhook endpoint. |

Supported event types: `host.synced`, `ssh_key.created`, `ssh_key.deleted`, `assignment.created`, `assignment.deleted`.

#### Attributes (computed)

| Attribute      | Type   | Description |
|----------------|--------|-------------|
| `id`           | string | UUID of the webhook endpoint. |
| `is_active`    | bool   | Whether the webhook endpoint is currently active. |
| `failure_count`| number | Cumulative delivery failure count. |
| `disabled_at`  | string | ISO 8601 timestamp when the endpoint was automatically disabled due to failures, or null. |
| `created_at`   | string | ISO 8601 creation timestamp. |
| `updated_at`   | string | ISO 8601 timestamp of the last update. |

#### Import

```shell
terraform import lockwave_webhook_endpoint.ops_alerts <endpoint-uuid>
```

---

## Data Sources

### `lockwave_team`

Fetches the Lockwave team identified by the provider's `team_id`.

```hcl
data "lockwave_team" "current" {}

output "team_name" {
  value = data.lockwave_team.current.name
}
```

#### Attributes

| Attribute      | Type   | Description |
|----------------|--------|-------------|
| `id`           | string | UUID of the team. |
| `name`         | string | Team name. |
| `personal_team`| bool   | Whether this is a personal (single-user) team. |
| `created_at`   | string | ISO 8601 creation timestamp. |

---

### `lockwave_host`

Fetches a single Lockwave host by ID.

```hcl
data "lockwave_host" "web01" {
  id = "550e8400-e29b-41d4-a716-446655440000"
}
```

#### Arguments

| Argument | Type   | Required | Description |
|----------|--------|----------|-------------|
| `id`     | string | Yes      | UUID of the host. |

#### Attributes

Returns all fields described under the `lockwave_host` resource (excluding `credential`), plus a computed `host_users` list.

---

### `lockwave_hosts`

Fetches all Lockwave hosts for the current team.

```hcl
data "lockwave_hosts" "all" {}
```

#### Attributes

| Attribute | Type              | Description |
|-----------|-------------------|-------------|
| `hosts`   | list(host object) | List of hosts. Each element exposes the same attributes as `lockwave_host` data source. |

---

### `lockwave_ssh_key`

Fetches a single Lockwave SSH key by ID.

```hcl
data "lockwave_ssh_key" "deploy" {
  id = "550e8400-e29b-41d4-a716-446655440001"
}
```

#### Arguments

| Argument | Type   | Required | Description |
|----------|--------|----------|-------------|
| `id`     | string | Yes      | UUID of the SSH key. |

#### Attributes

Returns all computed fields described under `lockwave_ssh_key` resource (excluding `private_key` and `mode`).

---

### `lockwave_ssh_keys`

Fetches all Lockwave SSH keys for the current team.

```hcl
data "lockwave_ssh_keys" "all" {}
```

#### Attributes

| Attribute | Type                | Description |
|-----------|---------------------|-------------|
| `keys`    | list(key object)    | List of SSH keys. Each element exposes the same attributes as `lockwave_ssh_key` data source. |

---

## Full Example

```hcl
terraform {
  required_providers {
    lockwave = {
      source  = "fwartner/lockwave"
      version = "~> 0.1"
    }
  }
}

provider "lockwave" {
  # api_url   = "https://lockwave.io"    # optional — defaults to https://lockwave.io
  api_token = var.lockwave_api_token
  team_id   = var.lockwave_team_id
}

variable "lockwave_api_token" {
  description = "Lockwave API token (Sanctum Bearer token)."
  type        = string
  sensitive   = true
}

variable "lockwave_team_id" {
  description = "UUID of the Lockwave team."
  type        = string
}

# ---------- Data sources ----------

data "lockwave_team" "current" {}

data "lockwave_hosts" "all" {}

data "lockwave_ssh_keys" "all" {}

# ---------- SSH Key (server-generated ed25519) ----------

resource "lockwave_ssh_key" "deploy" {
  name     = "deploy-key"
  mode     = "generate"
  key_type = "ed25519"
}

# Store the private key in a local file (for demo purposes).
resource "local_sensitive_file" "deploy_private_key" {
  content         = lockwave_ssh_key.deploy.private_key
  filename        = "${path.module}/deploy_key"
  file_permission = "0600"
}

# ---------- Host ----------

resource "lockwave_host" "web01" {
  display_name = "web-01"
  hostname     = "web-01.example.com"
  os           = "linux"
  arch         = "x86_64"
}

# ---------- Host User ----------

resource "lockwave_host_user" "ubuntu" {
  host_id = lockwave_host.web01.id
  os_user = "ubuntu"
}

# ---------- Assignment ----------

resource "lockwave_assignment" "deploy_to_web01" {
  ssh_key_id   = lockwave_ssh_key.deploy.id
  host_user_id = lockwave_host_user.ubuntu.id
}

# ---------- Webhook Endpoint ----------

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

# ---------- Outputs ----------

output "team_name" {
  value = data.lockwave_team.current.name
}

output "web01_id" {
  value = lockwave_host.web01.id
}

output "web01_credential" {
  value     = lockwave_host.web01.credential
  sensitive = true
}

output "deploy_key_fingerprint" {
  value = lockwave_ssh_key.deploy.fingerprint_sha256
}
```

---

## Development

### Building from source

```shell
git clone https://github.com/fwartner/terraform-provider-lockwave.git
cd terraform-provider-lockwave
go build -o terraform-provider-lockwave
```

### Running tests

Unit and provider tests (no live API required for unit tests):

```shell
go test ./...
```

Acceptance tests (require a live Lockwave account — see environment variables above):

```shell
TF_ACC=1 go test ./... -v
```

### Using a locally built provider

Create a `~/.terraformrc` override so Terraform uses your local binary instead of the registry version:

```hcl
provider_installation {
  dev_overrides {
    "fwartner/lockwave" = "/home/you/.terraform.d/plugins/registry.terraform.io/fwartner/lockwave/0.1.0/linux_amd64"
  }
  direct {}
}
```

Then run `make install` to build and copy the binary to that path.

### Makefile targets

| Target    | Description |
|-----------|-------------|
| `build`   | Compile the provider binary. |
| `test`    | Run all unit tests. |
| `testacc` | Run acceptance tests (requires `TF_ACC=1` and live credentials). |
| `lint`    | Run `golangci-lint`. |
| `fmt`     | Format all Go source files with `gofmt`. |
| `install` | Build and install the binary to `~/.terraform.d/plugins/`. |
| `clean`   | Remove the compiled binary. |

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

[lockwave.io](https://lockwave.io)
