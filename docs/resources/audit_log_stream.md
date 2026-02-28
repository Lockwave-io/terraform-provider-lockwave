---
page_title: "lockwave_audit_log_stream Resource - Lockwave"
subcategory: ""
description: |-
  Manages a Lockwave audit log stream that forwards audit events to an external destination (webhook or S3).
---

# lockwave_audit_log_stream (Resource)

Manages a Lockwave audit log stream that forwards audit events to an external destination. Supported destination types are `webhook` and `s3`.

## Example Usage

### Webhook Stream

```terraform
resource "lockwave_audit_log_stream" "webhook" {
  type = "webhook"

  config {
    url    = "https://siem.example.com/ingest/lockwave"
    secret = var.audit_webhook_secret
  }
}
```

### S3 Stream

```terraform
resource "lockwave_audit_log_stream" "s3" {
  type = "s3"

  config {
    bucket            = "my-audit-logs"
    region            = "eu-central-1"
    prefix            = "lockwave/"
    access_key_id     = var.aws_access_key_id
    secret_access_key = var.aws_secret_access_key
  }
}
```

## Schema

### Required

- `type` (String) - Stream destination type. One of: `webhook`, `s3`. Changing this forces a new resource.
- `config` (Block) - Type-specific configuration. Provide only the fields relevant to the chosen type.

### Config Block

**Webhook fields:**

- `url` (String, Optional) - HTTPS URL to deliver audit events to.
- `secret` (String, Sensitive, Optional) - Shared secret for HMAC signature verification.

**S3 fields:**

- `bucket` (String, Optional) - S3 bucket name.
- `region` (String, Optional) - AWS region of the S3 bucket.
- `prefix` (String, Optional) - Key prefix for uploaded audit log objects.
- `access_key_id` (String, Sensitive, Optional) - AWS access key ID.
- `secret_access_key` (String, Sensitive, Optional) - AWS secret access key.

### Read-Only

- `id` (String) - UUID of the audit log stream.
- `is_active` (Boolean) - Whether the audit log stream is currently active.
- `created_at` (String) - ISO 8601 creation timestamp.
- `updated_at` (String) - ISO 8601 timestamp of the last update.

## Import

```shell
terraform import lockwave_audit_log_stream.webhook <stream-uuid>
```

~> **Note:** Sensitive fields (`secret`, `access_key_id`, `secret_access_key`) are not returned by the API after creation. After import, these fields will be empty in state and must be set again.
