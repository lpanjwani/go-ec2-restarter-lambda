output "lambda_arn" {
  description = "ARN of the Lambda function"
  value       = aws_lambda_function.ec2_restarter.arn
}

output "lambda_role_arn" {
  description = "ARN of the Lambda IAM role"
  value       = aws_iam_role.lambda_role.arn
}

output "cloudwatch_rule_arn" {
  description = "ARN of the CloudWatch Event Rule"
  value       = aws_cloudwatch_event_rule.schedule.arn
}
