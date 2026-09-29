variable "aws_region" {
  type        = string
  description = "Region for the opspilot-aws-dev portfolio cluster. Set this explicitly."
}

variable "allowed_account_id" {
  type        = string
  description = "Only this 12-digit account may be planned or applied."

  validation {
    condition     = can(regex("^[0-9]{12}$", var.allowed_account_id)) && var.allowed_account_id != "000000000000"
    error_message = "allowed_account_id must be the real 12-digit portfolio account, not the example placeholder."
  }
}

variable "allowed_regions" {
  type        = list(string)
  description = "Regions this environment is allowed to use."

  validation {
    condition     = contains(var.allowed_regions, var.aws_region)
    error_message = "aws_region is not in allowed_regions."
  }
}

variable "environment" {
  type        = string
  description = "Must be dev. Production names are rejected."

  validation {
    condition     = var.environment == "dev"
    error_message = "Only environment=dev is allowed for this stack."
  }
}

variable "cluster_name" {
  type    = string
  default = "opspilot-aws-dev"

  validation {
    condition     = var.cluster_name == "opspilot-aws-dev"
    error_message = "The portfolio cluster name is fixed to opspilot-aws-dev."
  }
}

variable "kubernetes_version" {
  type    = string
  default = "1.35"

  validation {
    condition     = contains(["1.34", "1.35", "1.36"], var.kubernetes_version)
    error_message = "Use an EKS version in standard support. Extended-support versions such as 1.31 cost more."
  }
}

variable "vpc_cidr" {
  type    = string
  default = "10.42.0.0/16"
}

variable "instance_type" {
  type    = string
  default = "t3.medium"

  validation {
    condition     = contains(["t3.small", "t3.medium", "t3.large"], var.instance_type)
    error_message = "Instance type must stay in the portfolio set: t3.small, t3.medium, or t3.large."
  }
}

variable "node_min" {
  type    = number
  default = 1
}

variable "node_desired" {
  type    = number
  default = 1
}

variable "node_max" {
  type    = number
  default = 2

  validation {
    condition     = var.node_min >= 1 && var.node_min <= var.node_desired && var.node_desired <= var.node_max && var.node_max <= 3
    error_message = "Keep the node group between 1 and 3 instances, with min <= desired <= max."
  }
}

variable "api_cidrs" {
  type        = list(string)
  description = "CIDRs allowed to reach the public EKS API. Use your current public address as /32."

  validation {
    condition     = length(var.api_cidrs) > 0 && !contains(var.api_cidrs, "0.0.0.0/0") && !contains(var.api_cidrs, "::/0")
    error_message = "api_cidrs must name specific admin networks. 0.0.0.0/0 is refused."
  }
}

variable "ecr_repositories" {
  type    = list(string)
  default = ["opspilot-demo"]
}
