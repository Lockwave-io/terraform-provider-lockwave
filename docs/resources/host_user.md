---
page_title: "lockwave_host_user Resource - Lockwave"
subcategory: ""
description: |-
  Manages an OS user record on a Lockwave host. These records define which authorized_keys files the daemon manages.
---

# lockwave_host_user (Resource)

Manages an OS user record on a Lockwave host. These records define which `authorized_keys` files the daemon manages.

## Example Usage

```terraform
resource "lockwave_host_user" "ubuntu" {
  host_id = lockwave_host.web01.id
  os_user = "ubuntu"
}

resource "lockwave_host_user" "deploy" {
  host_id              = lockwave_host.web01.id
  os_user              = "deploy"
  authorized_keys_path = "/home/deploy/.ssh/authorized_keys"
}
```

## Schema

### Required

- `host_id` (String) - UUID of the parent host. Changing this forces a new resource.
- `os_user` (String) - OS username (e.g. `ubuntu`, `ec2-user`, `deploy`).

### Optional

- `authorized_keys_path` (String) - Absolute path to the `authorized_keys` file. Defaults to the standard location for the OS user. Computed if omitted.

### Read-Only

- `id` (String) - UUID of the host user.
- `created_at` (String) - ISO 8601 creation timestamp.

## Import

Import using the composite ID format `<host_id>/<user_id>`:

```shell
terraform import lockwave_host_user.ubuntu <host-uuid>/<user-uuid>
```
