output "aws_profile" {
  value = var.aws_profile
}

output "aws_region" {
  value = var.aws_region
}

output "cluster_name" {
  value = var.cluster_name
}

output "state_bucket_name" {
  value = var.state_bucket_name
}

output "k8s_version" {
  value = var.k8s_version
}

output "cluster_image" {
  value = var.cluster_image
}

output "k8s_api_ingress_sources" {
  value = var.k8s_api_ingress_sources
}

output "cluster_master_size" {
  value = var.cluster_master_size
}

output "cluster_masters_per_subnet" {
  value = var.cluster_masters_per_subnet
}

output "cluster_node_instance_groups" {
  value = var.cluster_node_instance_groups
}

output "cluster_vpc_cidr" {
  value = var.cluster_vpc_cidr
}

output "cluster_additional_policies_node" {
  value = var.cluster_additional_policies_node
}

output "aws_account_id" {
  value = data.aws_caller_identity.self.account_id
}

output "aws_iam_cluster_admin_role_arn" {
  value = aws_iam_role.cluster_admin.arn
}

output "cluster_vpc_id" {
  value = module.cluster_vpc.vpc_id
}

output "cluster_public_hosted_zone_id" {
  value = data.aws_route53_zone.public.zone_id
}

output "cluster_public_subnet_ids" {
  value = module.cluster_vpc.public_subnets
}

output "cluster_private_subnet_ids" {
  value = module.cluster_vpc.private_subnets
}

output "cluster_nat_gateway_ids" {
  value = module.cluster_vpc.natgw_ids
}

output "cluster_availability_zones" {
  value = module.cluster_vpc.azs
}

output "k8s_api_certificate_arn" {
  value = aws_acm_certificate.k8s_api.arn
}

output "aws_provider_default_tags" {
  value = var.aws_provider_default_tags
}
