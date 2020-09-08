aws_profile = "000000000000"
aws_region = "us-east-1"

cluster_vpc_cidr = "172.80.0.0/16"

private_subnets = [
  {
    cidr_block        = "172.80.0.0/24"
    availability_zone = "us-east-1c"
  }
]

public_subnets = [
  {
    cidr_block        = "172.80.10.0/24"
    availability_zone = "us-east-1c"
  }
]
