variable "aws_region" {
  description = "AWS region"
  type        = string
}

variable "aws_profile" {
  description = "AWS region"
  type        = string
}

variable "lambda_function_name" {
  description = "Name of the Lambda function"
  type        = string
}

variable "website_url" {
  description = "URL of the website to monitor"
  type        = string
}

variable "instance_id" {
  description = "ID of the EC2 instance to restart"
  type        = string
}

variable "schedule_expression" {
  description = "CloudWatch Events schedule expression"
  type        = string
  default     = "rate(15 minutes)"
}
