output "server_ip" {
  description = "Public IPv4 address of the server"
  value       = hcloud_server.main.ipv4_address
}

output "server_ipv6" {
  description = "Public IPv6 address of the server"
  value       = hcloud_server.main.ipv6_address
}

output "server_id" {
  description = "Hetzner server ID"
  value       = hcloud_server.main.id
}

resource "local_file" "ansible_inventory" {
  content  = "[servers]\n${hcloud_server.main.ipv4_address} ansible_user=${var.user_name} ansible_ssh_private_key_file=${var.ssh_private_key_path}\n"

  filename        = "${path.module}/../ansible/inventory.ini"
  file_permission = "0600"
}
