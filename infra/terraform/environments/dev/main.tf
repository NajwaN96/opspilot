module "network" {
  source = "../../modules/network"

  name     = var.cluster_name
  vpc_cidr = var.vpc_cidr
  az_count = 2
}

module "iam" {
  source = "../../modules/iam"

  name = var.cluster_name
}

module "eks" {
  source = "../../modules/eks"

  cluster_name              = var.cluster_name
  kubernetes_version        = var.kubernetes_version
  cluster_role_arn          = module.iam.cluster_role_arn
  node_role_arn             = module.iam.node_role_arn
  subnet_ids                = module.network.public_subnet_ids
  api_cidrs                 = var.api_cidrs
  instance_type             = var.instance_type
  node_min                  = var.node_min
  node_desired              = var.node_desired
  node_max                  = var.node_max
  cluster_policy_attachment = module.iam.cluster_policy_attachment
  node_policy_attachments   = module.iam.node_policy_attachments
}

module "ecr" {
  source = "../../modules/ecr"

  repositories = var.ecr_repositories
}
