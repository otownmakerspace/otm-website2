terraform {
  required_version = ">= 1.5, < 2.0"

  # NOTE: State is stored locally by default. For team use or CI/CD, configure
  # a remote backend (e.g., S3, GCS, or Terraform Cloud):
  #
  # backend "s3" {
  #   bucket = "your-terraform-state"
  #   key    = "otm-website/terraform.tfstate"
  #   region = "eu-central-1"
  # }

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.49"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

provider "hcloud" {
  token = var.hcloud_token
}
