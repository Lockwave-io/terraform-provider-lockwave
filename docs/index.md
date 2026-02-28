---
page_title: "Lockwave Provider"
subcategory: ""
description: |-
  The Lockwave provider manages SSH key lifecycle, hosts, assignments, and integrations via the Lockwave SaaS API.
---

# Lockwave Provider

The Lockwave provider lets you declaratively provision and enforce SSH access across your Linux/Unix fleet from within Terraform or OpenTofu. It manages SSH key lifecycle, hosts, assignments, and integrations via the [Lockwave](https://lockwave.io) SaaS API.

## Example Usage

```terraform
terraform {
  required_providers {
    lockwave = {
      source  = "lockwave-io/lockwave"
      version = "~> 0.1"
    }
  }
}

provider "lockwave" {
  api_token = var.lockwave_api_token
  team_id   = var.lockwave_team_id
}

resource "lockwave_ssh_key" "deploy" {
  name     = "deploy-key"
  mode     = "generate"
  key_type = "ed25519"
}

resource "lockwave_host" "web01" {
  display_name = "web-01"
  hostname     = "web-01.example.com"
  os           = "linux"
}

resource "lockwave_host_user" "ubuntu" {
  host_id = lockwave_host.web01.id
  os_user = "ubuntu"
}

resource "lockwave_assignment" "deploy_to_web01" {
  ssh_key_id   = lockwave_ssh_key.deploy.id
  host_user_id = lockwave_host_user.ubuntu.id
}
```

## Authentication

The provider requires a Sanctum Bearer token and a team UUID. These can be provided in the provider block or via environment variables.

### Environment Variables

| Environment Variable | Description |
|---------------------|-------------|
| `LOCKWAVE_API_URL`  | Base URL of the Lockwave API. Defaults to `https://lockwave.io`. |
| `LOCKWAVE_API_TOKEN`| Sanctum Bearer token for authentication. |
| `LOCKWAVE_TEAM_ID`  | UUID of the Lockwave team. |

## Schema

### Required

- `api_token` (String, Sensitive) - Sanctum Bearer token for authenticating with the Lockwave API. Can also be set via the `LOCKWAVE_API_TOKEN` environment variable.
- `team_id` (String) - UUID of the Lockwave team that all resources belong to. Can also be set via the `LOCKWAVE_TEAM_ID` environment variable.

### Optional

- `api_url` (String) - Base URL of the Lockwave API. Defaults to `https://lockwave.io`. Can also be set via the `LOCKWAVE_API_URL` environment variable.
