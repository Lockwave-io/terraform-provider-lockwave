---
page_title: "lockwave_notification_channel Resource - Lockwave"
subcategory: ""
description: |-
  Manages a Lockwave notification channel. Supported types are Slack and email.
---

# lockwave_notification_channel (Resource)

Manages a Lockwave notification channel. Supported types are `slack` and `email`.

## Example Usage

### Slack Channel

```terraform
resource "lockwave_notification_channel" "slack_ops" {
  type = "slack"
  name = "ops-alerts"

  config {
    webhook_url = var.slack_webhook_url
  }
}
```

### Email Channel

```terraform
resource "lockwave_notification_channel" "email_security" {
  type = "email"
  name = "security-team"

  config {
    recipients = [
      "security@example.com",
      "ops@example.com",
    ]
  }
}
```

## Schema

### Required

- `type` (String) - Channel type. One of: `slack`, `email`. Changing this forces a new resource.
- `name` (String) - Human-readable name for the notification channel.
- `config` (Block) - Type-specific configuration block. Populate only the fields relevant to the chosen type.

### Config Block

- `webhook_url` (String, Sensitive, Optional) - Incoming webhook URL (Slack channels only).
- `recipients` (List of String, Optional) - List of recipient email addresses (email channels only).

### Read-Only

- `id` (String) - UUID of the notification channel.
- `is_active` (Boolean) - Whether the notification channel is currently active.
- `created_at` (String) - ISO 8601 creation timestamp.
- `updated_at` (String) - ISO 8601 timestamp of the last update.

## Import

```shell
terraform import lockwave_notification_channel.slack_ops <channel-uuid>
```
