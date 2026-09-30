variable "package_path" {
  description = "Zip produced by package.sh. Required so apply cannot run against an empty function."
  type        = string
}

variable "expected_account_suffix" {
  description = "Last four digits of the only account this stack may touch."
  type        = string
  default     = "3851"
}
