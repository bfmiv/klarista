terraform {
  required_version = ">= 1.3.0"

  required_providers {
    aws = "~> 4.8"
  }
}

provider "aws" {
  profile = var.aws_profile
  region  = var.aws_region

  default_tags {
    tags = local.aws_provider_default_tags
  }
}

data "aws_caller_identity" "self" {
  provider = aws
}

data "aws_route53_zone" "public" {
  name         = "${local.cluster_dns_zone}."
  private_zone = false
}

resource "aws_acm_certificate" "k8s_api" {
  domain_name       = "api.${var.cluster_name}"
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "k8s_api_validation" {
  for_each = {
    for dvo in aws_acm_certificate.k8s_api.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }
  name    = each.value.name
  type    = each.value.type
  zone_id = data.aws_route53_zone.public.zone_id
  records = [each.value.record]
  ttl     = "60"
}

resource "aws_acm_certificate_validation" "k8s_api" {
  certificate_arn         = aws_acm_certificate.k8s_api.arn
  validation_record_fqdns = [for record in aws_route53_record.k8s_api_validation : record.fqdn]
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
}

module "cluster_vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.16.1"

  name                   = var.cluster_name
  cidr                   = var.cluster_vpc_cidr
  azs                    = distinct([for s in concat(var.private_subnets, var.public_subnets) : s.availability_zone])
  private_subnets        = [for s in var.private_subnets : s.cidr_block]
  public_subnets         = [for s in var.public_subnets : s.cidr_block]
  enable_dns_support     = true
  enable_dns_hostnames   = true
  enable_nat_gateway     = true
  one_nat_gateway_per_az = true
  reuse_nat_ips          = length(var.nat_elastic_ip_ids) > 0
  external_nat_ip_ids    = var.nat_elastic_ip_ids

  tags = {
    // This is so kops knows that the VPC resources can be used for k8s
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
  }

  // Tags required by k8s to launch services on the right subnets
  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
    "SubnetType"                      = "Private"
  }

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
    "SubnetType"             = "Public"
  }
}
