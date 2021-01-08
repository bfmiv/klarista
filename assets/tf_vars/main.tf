variable "aws_profile" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "encryption_key_arn" {
  type    = string
  default = null
}

output "aws_profile" {
  value = var.aws_profile
}

output "aws_region" {
  value = var.aws_region
}

output "encryption_key_arn" {
  value = var.encryption_key_arn
}
