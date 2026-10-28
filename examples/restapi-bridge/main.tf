terraform {
  required_providers {
    subprocess = {
      source  = "registry.terraform.io/alexanderankin/subprocess"
      version = "~> 0.1.0"
    }
    restapi = {
      source  = "Mastercard/restapi"
      version = "~> 3.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.14"
    }
  }
}

variable "bridge_port" {
  type    = number
  default = 8080
}

provider "restapi" {
  uri                  = "http://127.0.0.1:${var.bridge_port}"
  write_returns_object = true
}

ephemeral "subprocess_process" "bridge" {
  command = ["my-api-bridge", "--listen", "127.0.0.1:${var.bridge_port}"]
  # wait_timeout_ms = 10000

  wait_for_listening_port {
    port     = var.bridge_port
    protocol = "tcp"
  }
}

resource "time_sleep" "bridge_startup" {
  depends_on      = [ephemeral.subprocess_process.bridge]
  create_duration = "10s"
}

resource "restapi_object" "example" {
  path = "/api/widgets"
  data = jsonencode({
    name = "example-widget"
  })

  depends_on = [time_sleep.bridge_startup]
}
