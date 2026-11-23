terraform {
  required_providers {
    proxmox = {
      source  = "bpg/proxmox"
      version = "~> 0.69"
    }
  }
}

provider "proxmox" {
  endpoint  = "https://${var.proxmox_host}:8006/"
  api_token = "candidate3@pve!exam=${var.proxmox_api_token_secret}"
  insecure  = true
  ssh {
    agent = true
  }
}
