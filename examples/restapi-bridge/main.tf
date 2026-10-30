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
  }
}

variable "bridge_port" {
  type    = number
  default = 8080
}

// start the service
ephemeral "subprocess_process" "bridge" {
  command                   = ["my-api-bridge", "--listen", "127.0.0.1:${var.bridge_port}"]
  inputs                    = { uri = "http://127.0.0.1:${var.bridge_port}" }
  wait_for_startup_delay_ms = 10000

  wait_for_listening_port {
    port     = var.bridge_port
    protocol = "tcp"
  }
}

// then use the service
provider "restapi" {
  uri                  = ephemeral.subprocess_process.bridge.outputs["uri"]
  write_returns_object = true
}

resource "restapi_object" "example" {
  path = "/api/widgets"
  data = jsonencode({
    name = "example-widget"
  })
}
