terraform {
  required_version = ">= 1.3.0"

  required_providers {
    aws = "~> 4.8"
  }
}

variable "aws_profile" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "aws_provider_default_tags" {
  type    = any
  default = null
}

variable "cluster_name" {
  type = string
}

variable "state_bucket_name" {
  type = string
}

provider "aws" {
  profile = var.aws_profile
  region  = var.aws_region

  default_tags {
    tags = var.aws_provider_default_tags
  }
}

resource "aws_s3_bucket" "klarista_state" {
  bucket        = var.state_bucket_name
  acl           = "private"
  force_destroy = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    enabled = true
    noncurrent_version_expiration {
      days = 30
    }
  }

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "aws:kms"
      }
    }
  }

  tags = {
    Name = var.state_bucket_name
  }
}

resource "aws_s3_bucket_public_access_block" "klarista_state" {
  bucket                  = aws_s3_bucket.klarista_state.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

output "aws_profile" {
  value = var.aws_profile
}

output "aws_region" {
  value = var.aws_region
}
