locals {
  vms = {
    c3-docker-host  = { cores = 4, memory = 4096, role = "Application Host (Docker)" }
    c3-k8s-master   = { cores = 4, memory = 4096, role = "Kubernetes Control Plane" }
    c3-k8s-worker-1 = { cores = 4, memory = 4096, role = "Kubernetes Worker" }
    c3-k8s-worker-2 = { cores = 4, memory = 4096, role = "Kubernetes Worker" }
  }
}

resource "proxmox_virtual_environment_vm" "exam" {
  for_each = local.vms

  name      = each.key
  node_name = "Server1"
  pool_id   = "exam-c3"

  clone {
    vm_id = 9003
  }

  cpu {
    cores = each.value.cores
  }

  memory {
    dedicated = each.value.memory
  }

  disk {
    interface    = "scsi0"
    datastore_id = "local-lvm"
    size         = 30
  }

  initialization {
    datastore_id = "local-lvm"
    user_account {
      username = var.vm_username
      password = var.vm_password
      keys     = compact([var.ssh_public_key])
    }
    ip_config {
      ipv4 {
        address = "dhcp"
      }
    }
  }

  description = each.value.role

  agent {
    enabled = true
  }

  lifecycle {
    ignore_changes = [clone]
  }
}
