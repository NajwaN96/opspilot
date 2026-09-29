terraform {
  required_version = ">= 1.8.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.100"
    }
  }

  # Local state avoids creating an S3 bucket before the portfolio cluster exists.
  # docs/adr/0022-terraform-layout-and-state.md describes the remote-state move.
  backend "local" {
    path = "terraform.tfstate"
  }
}
