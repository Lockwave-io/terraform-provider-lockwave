---
page_title: "lockwave_team Data Source - Lockwave"
subcategory: ""
description: |-
  Fetches the Lockwave team identified by the provider's team_id.
---

# lockwave_team (Data Source)

Fetches the Lockwave team identified by the provider's `team_id`.

## Example Usage

```terraform
data "lockwave_team" "current" {}

output "team_name" {
  value = data.lockwave_team.current.name
}
```

## Schema

### Read-Only

- `id` (String) - UUID of the team.
- `name` (String) - Team name.
- `personal_team` (Boolean) - Whether this is a personal (single-user) team.
- `created_at` (String) - ISO 8601 creation timestamp.
