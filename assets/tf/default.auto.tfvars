aws_profile = null
aws_region = null
aws_authorized_accounts = []

# See https://github.com/kubernetes/kops/blob/master/docs/operations/images.md
cluster_image = "ubuntu/ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-20210621"
k8s_version = "1.20.8"

k8s_api_ingress_sources = []

cluster_master_size = "t3.large"

cluster_node_instance_groups = [
  {
    metadata = {
      name = "nodes"
    }
    spec = {
      machineType = "t3.xlarge"
      minSize     = 3
      maxSize     = 9
    }
  }
]

cluster_vpc_cidr = null

private_subnets = []

public_subnets = []
