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

variable "cluster_name" {
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

variable "cluster_node_instance_groups" {
  type = list(object({
    metadata = object({
      name = string
    })
    spec = object({
      machineType = string
      minSize     = number
      maxSize     = number
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
