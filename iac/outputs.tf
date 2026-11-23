output "vm_ips" {
  description = "DHCP IPv4 addresses reported by each VM's QEMU guest agent"
  value = {
    for name, vm in proxmox_virtual_environment_vm.exam :
    name => try(
      [for address in flatten(vm.ipv4_addresses) : address if address != "" && !startswith(address, "127.")][0],
      "address-pending"
    )
  }
}
