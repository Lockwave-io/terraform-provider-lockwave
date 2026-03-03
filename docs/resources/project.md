---
page_title: "lockwave_project Resource - Lockwave Provider"
subcategory: ""
description: |-
  Manages a Lockwave project. Projects are optional groupings for hosts and SSH keys within a team.
---

# lockwave_project (Resource)

Manages a Lockwave project. Projects are optional groupings for hosts and SSH keys within a team.

## Example Usage

```hcl
resource "lockwave_project" "production" {
  name        = "Production"
  description = "Production infrastructure hosts and keys"
  color       = "#EF4444"
}

resource "lockwave_host" "web" {
  display_name = "web-01"
  hostname     = "10.0.1.10"
  os           = "linux"
  project_id   = lockwave_project.production.id
}
```

## Argument Reference

- `name` - (Required) Name of the project.
- `description` - (Optional) Description of the project.
- `color` - (Optional) Hex color for the project badge (e.g. `#3B82F6`).

## Attribute Reference

- `id` - UUID of the project.
- `slug` - URL-friendly slug of the project (auto-generated from name).
- `created_at` - ISO 8601 timestamp of when the project was created.
- `updated_at` - ISO 8601 timestamp of when the project was last updated.

## Import

Projects can be imported using the project UUID:

```shell
terraform import lockwave_project.production <project-uuid>
```
