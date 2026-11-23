variable "proxmox_host" {
  type    = string
  default = "192.168.160.16"
}

variable "proxmox_api_token_secret" {
  type      = string
  sensitive = true
}

variable "vm_username" {
  type    = string
  default = "qt"
}

variable "vm_password" {
  type      = string
  sensitive = true
}

variable "ssh_public_key" {
  type    = string
  default = ""
}
