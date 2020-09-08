locals {
  aws_iam_admin_role_name = "Kubernetes.${var.cluster_name}.Admin"
  cluster_name_segments   = split(".", var.cluster_name)
  cluster_stage           = split("-", local.cluster_name_segments[0])[0]
  cluster_dns_zone        = join(".", slice(local.cluster_name_segments, 1, length(local.cluster_name_segments)))
  index_to_az             = ["c", "a", "b", "d", "e", "f"]
  kops_state_bucket       = "${replace(var.cluster_name, ".", "-")}-kops-state"
  tags = {
    environment = var.cluster_name
    terraform   = true
    workspace   = terraform.workspace
  }
}
