resource "hcloud_ssh_key" "default" {
  name       = "${var.server_name}-key"
  public_key = file(var.ssh_public_key_path)
}

resource "hcloud_server" "main" {
  name        = var.server_name
  server_type = var.server_type
  location    = var.location
  image       = var.image
  ssh_keys    = [hcloud_ssh_key.default.id]

  user_data = <<-CLOUDINIT
    #cloud-config
    hostname: ${var.server_name}
    manage_etc_hosts: true

    users:
      - name: ${var.user_name}
        gecos: "${var.user_name}"
        groups: [adm, sudo]
        shell: /bin/bash
        lock_passwd: false
        passwd: ${var.user_password_hash}
        sudo: ["ALL=(ALL) NOPASSWD:ALL"]
        ssh_authorized_keys:
          - ${trimspace(file(var.ssh_public_key_path))}

    package_update: true
    package_upgrade: true
    packages:
      - python3

    runcmd:
      - logger -t otm "[0.1] Cloud-init - starting provisioning"
      - mkdir -p /home/${var.user_name}/.ssh
      - chmod 700 /home/${var.user_name}/.ssh
      - echo '${trimspace(file(var.ssh_public_key_path))}' > /home/${var.user_name}/.ssh/authorized_keys
      - chmod 600 /home/${var.user_name}/.ssh/authorized_keys
      - chown -R ${var.user_name}:${var.user_name} /home/${var.user_name}/.ssh
      - logger -t otm "[0.2] Cloud-init - SSH directory configured"
      - logger -t otm "[0.3] Cloud-init - complete, handing off to Ansible"
  CLOUDINIT

  # Cloud-init runs exactly once at first boot — the SSH key it writes to
  # authorized_keys is the bootstrap key. Subsequent key rotations are managed
  # by Ansible (see roles/authorized_keys), so ignore drift on these fields
  # to prevent Terraform from destroying the server when the local pubkey
  # file changes.
  lifecycle {
    ignore_changes = [
      user_data,
      ssh_keys,
    ]
  }
}
