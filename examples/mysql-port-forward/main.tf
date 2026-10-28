terraform {
  required_providers {
    subprocess = {
      source  = "registry.terraform.io/alexanderankin/subprocess"
      version = "~> 0.1.0"
    }
    mysql = {
      source  = "petoju/mysql"
      version = "~> 3.0"
    }
  }
}

//<editor-fold desc="variables">
variable "ssh_host" {
  type = string
}

variable "ssh_user" {
  type = string
}

variable "mysql_host" {
  type = string
}

variable "mysql_port" {
  type    = number
  default = 3306
}

variable "local_port" {
  type    = number
  default = 13306
}

variable "mysql_admin_user" {
  type = string
}

variable "mysql_admin_password" {
  type      = string
  sensitive = true
}

variable "mysql_app_user" {
  type      = string
  sensitive = true
}

variable "mysql_app_password" {
  type      = string
  sensitive = true
}
//</editor-fold>

ephemeral "subprocess_process" "mysql_tunnel" {
  command = [
    "ssh", "-N", "-o", "BatchMode=yes", "-o", "ExitOnForwardFailure=yes",
    "-L", "127.0.0.1:${var.local_port}:${var.mysql_host}:${var.mysql_port}",
    "${var.ssh_user}@${var.ssh_host}",
  ]
  startup_delay_ms = 1000
}

provider "mysql" {
  endpoint = "127.0.0.1:${var.local_port}"
  username = var.mysql_admin_user
  password = var.mysql_admin_password
}

resource "mysql_user" "app" {
  user               = var.mysql_app_user
  host               = "%"
  plaintext_password = var.mysql_app_password

  depends_on = [ephemeral.subprocess_process.mysql_tunnel]
}
