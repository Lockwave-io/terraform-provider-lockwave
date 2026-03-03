---
page_title: "lockwave_projects Data Source - Lockwave Provider"
subcategory: ""
description: |-
  Fetches all Lockwave projects for the current team.
---

# lockwave_projects (Data Source)

Fetches all Lockwave projects for the current team.

## Example Usage

```hcl
data "lockwave_projects" "all" {}

output "project_names" {
  value = [for p in data.lockwave_projects.all.projects : p.name]
}
```

## Attribute Reference

- `projects` - List of projects. Each project has:
  - `id` - UUID of the project.
  - `name` - Name of the project.
  - `slug` - URL-friendly slug.
  - `description` - Description.
  - `color` - Hex color.
  - `created_at` - ISO 8601 creation timestamp.
  - `updated_at` - ISO 8601 last-updated timestamp.
