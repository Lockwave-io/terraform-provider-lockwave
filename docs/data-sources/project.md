---
page_title: "lockwave_project Data Source - Lockwave Provider"
subcategory: ""
description: |-
  Fetches a single Lockwave project by ID.
---

# lockwave_project (Data Source)

Fetches a single Lockwave project by ID.

## Example Usage

```hcl
data "lockwave_project" "prod" {
  id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

output "project_name" {
  value = data.lockwave_project.prod.name
}
```

## Argument Reference

- `id` - (Required) UUID of the project to look up.

## Attribute Reference

- `name` - Name of the project.
- `slug` - URL-friendly slug of the project.
- `description` - Description of the project.
- `color` - Hex color of the project badge.
- `created_at` - ISO 8601 creation timestamp.
- `updated_at` - ISO 8601 last-updated timestamp.
