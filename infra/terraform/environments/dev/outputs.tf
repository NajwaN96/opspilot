output "account_constraint" {
  value = var.allowed_account_id
}

output "region" {
  value = var.aws_region
}

output "cluster_name" {
  value = module.eks.cluster_name
}

output "cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "ecr_repository_urls" {
  value = module.ecr.repository_urls
}

output "vpc_id" {
  value = module.network.vpc_id
}
