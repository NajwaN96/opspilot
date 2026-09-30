locals {
  function_name = "opspilot-portfolio"
  log_group     = "/aws/lambda/opspilot-portfolio"
}

resource "aws_cloudwatch_log_group" "portfolio" {
  name              = local.log_group
  retention_in_days = 1
}

resource "aws_iam_role" "portfolio" {
  name = "opspilot-portfolio-lambda"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "portfolio_logs" {
  name = "opspilot-portfolio-logs"
  role = aws_iam_role.portfolio.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "logs:CreateLogStream",
        "logs:PutLogEvents",
      ]
      Resource = "arn:aws:logs:us-east-1:${data.aws_caller_identity.current.account_id}:log-group:${local.log_group}:*"
    }]
  })
}

resource "aws_lambda_function" "portfolio" {
  function_name    = local.function_name
  role             = aws_iam_role.portfolio.arn
  runtime          = "python3.12"
  architectures    = ["arm64"]
  handler          = "handler.handler"
  filename         = var.package_path
  source_code_hash = filebase64sha256(var.package_path)
  memory_size      = 128
  timeout          = 10
  publish          = false

  ephemeral_storage {
    size = 512
  }

  logging_config {
    log_format            = "JSON"
    application_log_level = "FATAL"
    system_log_level      = "WARN"
    log_group             = aws_cloudwatch_log_group.portfolio.name
  }

  environment {
    variables = {
      OPSPILOT_RUNTIME = "aws-portfolio-demo"
    }
  }

  depends_on = [aws_iam_role_policy.portfolio_logs, aws_cloudwatch_log_group.portfolio]

  lifecycle {
    precondition {
      condition     = endswith(data.aws_caller_identity.current.account_id, var.expected_account_suffix) && data.aws_region.current.region == "us-east-1"
      error_message = "Refusing to create the portfolio function outside account suffix 3851 in us-east-1."
    }
  }
}

resource "aws_lambda_function_url" "portfolio" {
  function_name      = aws_lambda_function.portfolio.function_name
  authorization_type = "NONE"
}

resource "aws_lambda_permission" "url" {
  statement_id           = "FunctionURLAllowPublic"
  action                 = "lambda:InvokeFunctionUrl"
  function_name          = aws_lambda_function.portfolio.function_name
  principal              = "*"
  function_url_auth_type = "NONE"
}

resource "aws_lambda_permission" "invoke" {
  statement_id             = "FunctionURLInvokeAllowPublic"
  action                   = "lambda:InvokeFunction"
  function_name            = aws_lambda_function.portfolio.function_name
  principal                = "*"
  invoked_via_function_url = true
}
