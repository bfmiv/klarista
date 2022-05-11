variable "aws_authorized_accounts" {
  type    = list(string)
  default = []
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
  validation {
    condition     = length(var.cluster_name) <= 44
    error_message = "Cluster names must be 44 characters or less."
  }
}

variable "state_bucket_name" {
  type = string
}

variable "k8s_version" {
  type = string
}

variable "cluster_image" {
  type = string
}

variable "k8s_api_ingress_sources" {
  type = list(string)
}

variable "cluster_master_size" {
  type = string
}

variable "cluster_masters_per_subnet" {
  type    = number
  default = 1
}

variable "cluster_node_instance_groups" {
  type = list(object({
    availability_zones = optional(list(string))
    metadata = object({
      name = string
    })
    spec = object({
      machineType = string
      minSize     = number
      maxSize     = number
      nodeLabels  = optional(any)
      taints = optional(list(object({
        effect = string
        key    = string
        value  = string
      })))
      volumes = optional(list(object({
        device    = string
        size      = number
        type      = string
        encrypted = optional(bool)
        key       = optional(string)
      })))
    })
  }))
}

variable "cluster_vpc_cidr" {
  type = string
}

variable "cluster_additional_policies_node" {
  type = list(object({
    Effect   = string
    Action   = list(string)
    Resource = list(string)
  }))
  description = "Additional IAM policies for the cluster nodes role"
  default     = []
}

variable "encryption_key_arn" {
  type    = string
  default = null
}

variable "public_subnets" {
  type = list(object({
    cidr_block        = string
    availability_zone = string
  }))

  validation {
    condition     = length(var.public_subnets) >= 1
    error_message = "You must define at least one public subnet."
  }
}

variable "private_subnets" {
  type = list(object({
    cidr_block        = string
    availability_zone = string
  }))

  validation {
    condition     = length(var.private_subnets) >= 1
    error_message = "You must define at least one private subnet."
  }
}

variable "nat_elastic_ip_ids" {
  description = "IDs of pre-allocated elastic IP addresses to be associated with NAT gateways"
  type        = list(string)
  default     = []
}

locals {
  aws_iam_admin_role_name = "K8S.${var.cluster_name}.ClusterAdmin"
  cluster_name_segments   = split(".", var.cluster_name)
  cluster_stage           = split("-", local.cluster_name_segments[0])[0]
  cluster_dns_zone        = join(".", slice(local.cluster_name_segments, 1, length(local.cluster_name_segments)))
  index_to_az             = ["a", "b", "c", "d", "e", "f"]
  aws_provider_default_tags = (
    var.aws_provider_default_tags != null
    ? { for key, value in var.aws_provider_default_tags : key => tostring(value) }
    : {
      environment = var.cluster_name
      terraform   = "true"
      workspace   = terraform.workspace
    }
  )
}
