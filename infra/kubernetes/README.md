# Kubernetes manifests

The workload YAML lives in `gitops/base`. The local cluster applies `gitops/overlays/local`. The AWS dev cluster is `gitops/overlays/aws-dev`, reconciled by Argo CD after the cloud approval gate.
