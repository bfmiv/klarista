terraform {
  required_version = ">= 0.12"

  required_providers {
    aws = "~> 2.69"
  }
}

provider "aws" {
  profile = var.aws_profile
  region  = var.aws_region
}

data "aws_caller_identity" "self" {
  provider = aws
}

data "aws_route53_zone" "public" {
  name = "${local.cluster_dns_zone}."
}

resource "aws_s3_bucket" "kops_state" {
  bucket        = local.kops_state_bucket
  acl           = "private"
  force_destroy = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    enabled = true

    noncurrent_version_expiration {
      days = 90
    }
  }

  tags = merge(local.tags, {
    Name = local.kops_state_bucket
  })
}

resource "aws_s3_bucket_public_access_block" "kops_state" {
  bucket                  = aws_s3_bucket.kops_state.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_acm_certificate" "k8s_api" {
  domain_name       = "api.${var.cluster_name}"
  validation_method = "DNS"

  tags = merge(local.tags)

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "k8s_api_validation" {
  name    = aws_acm_certificate.k8s_api.domain_validation_options[0].resource_record_name
  type    = aws_acm_certificate.k8s_api.domain_validation_options[0].resource_record_type
  zone_id = data.aws_route53_zone.public.zone_id
  records = [aws_acm_certificate.k8s_api.domain_validation_options[0].resource_record_value]
  ttl     = "60"
}

resource "aws_acm_certificate_validation" "k8s_api" {
  certificate_arn = aws_acm_certificate.k8s_api.arn
  validation_record_fqdns = [
    aws_route53_record.k8s_api_validation.fqdn
  ]
}

resource "aws_iam_role" "cluster_admin" {
  name = local.aws_iam_admin_role_name

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = [for id in distinct(concat([data.aws_caller_identity.self.account_id], var.aws_authorized_accounts)) : "arn:aws:iam::${id}:root"]
        }
      }
    ]
  })

  tags = merge(local.tags)
}

module "cluster_vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.44.0"

  name                   = var.cluster_name
  cidr                   = var.cluster_vpc_cidr
  azs                    = distinct([for s in concat(var.private_subnets, var.public_subnets) : s.availability_zone])
  private_subnets        = [for s in var.private_subnets : s.cidr_block]
  public_subnets         = [for s in var.public_subnets : s.cidr_block]
  enable_dns_support     = true
  enable_dns_hostnames   = true
  enable_nat_gateway     = true
  one_nat_gateway_per_az = true

  tags = merge(local.tags, {
    // This is so kops knows that the VPC resources can be used for k8s
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
  })

  // Tags required by k8s to launch services on the right subnets
  private_subnet_tags = merge(local.tags, {
    "kubernetes.io/role/internal-elb" = 1
    "SubnetType"                      = "Private"
  })

  public_subnet_tags = merge(local.tags, {
    "kubernetes.io/role/elb" = 1
    "SubnetType"             = "Public"
  })
}
