variable "hcloud_token" {
  description = "Hetzner Cloud API token (read+write)"
  type        = string
  sensitive   = true
}

variable "server_name" {
  description = "Name for the Hetzner server"
  type        = string
  default     = "otm-staging"
}

variable "server_type" {
  description = "Hetzner server type (e.g. cpx21, cx22)"
  type        = string
  default     = "cpx32"
}

variable "location" {
  description = "Hetzner datacenter location (e.g. hel1, fsn1, nbg1)"
  type        = string
  default     = "fsn1"
}

variable "image" {
  description = "OS image for the server"
  type        = string
  default     = "ubuntu-24.04"
}

variable "ssh_public_key_path" {
  description = "Path to the local SSH public key to upload"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path to the local SSH private key (for Ansible inventory)"
  type        = string
}

variable "user_name" {
  description = "Linux user to create on the server"
  type        = string
}

variable "user_password_hash" {
  description = "SHA-512 hashed password for the user (generate with: openssl passwd -6)"
  type        = string
  sensitive   = true
}
