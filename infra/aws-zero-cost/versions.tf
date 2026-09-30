terraform {
  required_version = ">= 1.8.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.28"
    }
  }

  # Local state only. An S3 backend would spend credits, which this stack refuses.
  backend "local" {
    path = "terraform.tfstate"
  }
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      Project     = "opspilot"
      Environment = "aws-portfolio-demo"
      CostPolicy  = "zero-out-of-pocket"
    }
  }
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}
