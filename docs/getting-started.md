# Getting Started

Guide to getting started with the provider.

## Installation

Install the provider:

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-$p:latest
```

## Prerequisites

- Kubernetes cluster with Crossplane installed

## Quick Start

1. Create a ProviderConfig:

```yaml
apiVersion: $p.crossplane.io/v1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: provider-credentials
      namespace: crossplane-system
```
