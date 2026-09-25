# Getting Started

Guide to getting started with provider-gitea.

## Installation

Install the provider:

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-gitea:v0.15.9
```

## Prerequisites

- Kubernetes cluster with Crossplane v2.5.0 or later installed

## Quick Start

1. Create a ProviderConfig:

```yaml
apiVersion: gitea.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: gitea-credentials
      namespace: crossplane-system
```
