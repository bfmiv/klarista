# klarista

> Hot clusters brewed daily!

`klarista` is a command line tool that generates terraform modules for kops clusters.

## Prerequisites

- [`kops@>=1.22`](https://kubernetes.io/docs/setup/production-environment/tools/kops/)
- [`kubectl@>=1.23`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [`terraform@>=1.3.0`](https://www.terraform.io/downloads.html)

## Installation

### Precompiled Binary

```bash
docker run --rm bernardmcmanus/klarista install | bash

# Usage:
# docker run --rm bernardmcmanus/klarista[:version = latest] install [path = /usr/local/bin] | bash
```

### Build from source

```bash
make install
```

## Getting Started

```bash
export CLUSTER=dev2-lavender.bfmiv.com

cd test/fixtures/$CLUSTER

klarista create $CLUSTER --yes

export KUBECONFIG=`klarista get $CLUSTER kubeconfig.yaml --path`

kubectl get pod -A

klarista destroy $CLUSTER --yes

unset CLUSTER KUBECONFIG
```
