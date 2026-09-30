output "portfolio_url" {
  description = "Public HTTPS URL for the portfolio console."
  value       = aws_lambda_function_url.portfolio.function_url
}

output "function_name" {
  value = aws_lambda_function.portfolio.function_name
}

output "region" {
  value = data.aws_region.current.region
}

output "account_masked" {
  value = "********${substr(data.aws_caller_identity.current.account_id, 8, 4)}"
}
