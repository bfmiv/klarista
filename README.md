# klarista

> Hot clusters brewed daily!

`klarista` is a command line tool that generates terraform modules for kops clusters.

## Prerequisites

- [`kops@>=1.18`](https://kubernetes.io/docs/setup/production-environment/tools/kops/)
- [`kubectl@>=1.18`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [`terraform@>=0.13`](https://www.terraform.io/downloads.html)

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

export KUBECONFIG=`klarista get $CLUSTER kubeconfig.yaml`

kubectl get pod --all-namespaces

klarista destroy $CLUSTER --yes

unset CLUSTER KUBECONFIG
```
