---
page_title: "lockwave_assignment Resource - Lockwave"
subcategory: ""
description: |-
  Manages a Lockwave assignment that grants an SSH key access to an OS user on a host.
---

# lockwave_assignment (Resource)

Manages a Lockwave assignment that grants an SSH key access to an OS user on a host. All input fields are ForceNew — any change recreates the assignment.

## Example Usage

```terraform
resource "lockwave_assignment" "deploy_to_web01" {
  ssh_key_id   = lockwave_ssh_key.deploy.id
  host_user_id = lockwave_host_user.ubuntu.id
}

resource "lockwave_assignment" "temporary_access" {
  ssh_key_id   = lockwave_ssh_key.deploy.id
  host_user_id = lockwave_host_user.ubuntu.id
  expires_at   = "2025-12-31T23:59:59Z"
}
```

## Schema

### Required

- `ssh_key_id` (String) - UUID of the SSH key to assign. Changing this forces a new resource.
- `host_user_id` (String) - UUID of the host user to assign the key to. Changing this forces a new resource.

### Optional

- `expires_at` (String) - Optional ISO 8601 expiry timestamp for the assignment. Computed if omitted.

### Read-Only

- `id` (String) - UUID of the assignment.
- `created_at` (String) - ISO 8601 creation timestamp.

## Import

```shell
terraform import lockwave_assignment.deploy_to_web01 <assignment-uuid>
```
