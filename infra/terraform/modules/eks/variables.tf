variable "cluster_name" {
  type = string
}

variable "kubernetes_version" {
  type = string
}

variable "cluster_role_arn" {
  type = string
}

variable "node_role_arn" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "api_cidrs" {
  type = list(string)
}

variable "instance_type" {
  type = string
}

variable "node_min" {
  type = number
}

variable "node_desired" {
  type = number
}

variable "node_max" {
  type = number
}

variable "cluster_policy_attachment" {
  type = string
}

variable "node_policy_attachments" {
  type = list(string)
}
