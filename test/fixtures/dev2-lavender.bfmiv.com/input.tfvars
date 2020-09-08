aws_profile = "000000000000"
aws_region = "us-east-1"

cluster_node_instance_groups = [
  {
    metadata = {
      name = "nodes.t3.large"
    }
    spec = {
      machineType = "t3.large"
      minSize     = 3
      maxSize     = 9
    }
  },
  {
    metadata = {
      name = "nodes.t3.xlarge"
    }
    spec = {
      machineType = "t3.xlarge"
      minSize     = 3
      maxSize     = 6
    }
  }
]

cluster_vpc_cidr = "172.70.0.0/24"

private_subnets = [
  {
    cidr_block        = "172.70.0.0/27"
    availability_zone = "us-east-1a"
  },
  {
    cidr_block        = "172.70.0.32/27"
    availability_zone = "us-east-1b"
  },
  {
    cidr_block        = "172.70.0.64/27"
    availability_zone = "us-east-1c"
  }
]

public_subnets = [
  {
    cidr_block        = "172.70.0.96/27"
    availability_zone = "us-east-1a"
  },
  {
    cidr_block        = "172.70.0.128/27"
    availability_zone = "us-east-1b"
  },
  {
    cidr_block        = "172.70.0.160/27"
    availability_zone = "us-east-1c"
  }
]
