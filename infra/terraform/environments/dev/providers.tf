provider "aws" {
  region              = var.aws_region
  allowed_account_ids = [var.allowed_account_id]

  default_tags {
    tags = {
      Project     = "OpsPilot"
      Environment = var.environment
      ManagedBy   = "OpenTofu"
      Purpose     = "portfolio"
    }
  }
}
