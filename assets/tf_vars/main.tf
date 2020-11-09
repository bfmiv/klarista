variable "aws_profile" {
  type = string
}

variable "aws_region" {
  type = string
}

output "aws_profile" {
  value = var.aws_profile
}

output "aws_region" {
  value = var.aws_region
}
