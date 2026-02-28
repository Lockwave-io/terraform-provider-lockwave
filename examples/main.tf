terraform {
  required_providers {
    lockwave = {
      source  = "lockwave-io/lockwave"
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

# ---------- Notification Channel (Slack) ----------

resource "lockwave_notification_channel" "slack_ops" {
  type = "slack"
  name = "ops-slack"
  config = {
    webhook_url = "https://hooks.slack.com/services/T00/B00/xxx"
  }
}

# ---------- Audit Log Stream (Webhook) ----------

resource "lockwave_audit_log_stream" "siem" {
  type = "webhook"
  config = {
    url    = "https://siem.example.com/ingest/lockwave"
    secret = "whsec_supersecret"
  }
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
