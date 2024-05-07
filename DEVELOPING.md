# Developing for Contribution

Oracle welcomes your contributions! For more information about how to contribute, review the [Contributing](CONTRIBUTING.md) guidelines.

The below is guidance for establishing a development environment and building/testing contributions before submitting them for review.

## Prerequisites

This Operator is developed using the [Operator SDK](https://sdk.operatorframework.io/) Framework.  It is recommended to walk-through the [Operator SDK Go tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/) to familiarise yourself with the process.

- Access to a Kubernetes v1.28.8+ cluster.
- Access to a Container Registry.
- kubectl version v1.29.3+.
- go version v1.21.9+.
- docker or podman.  If using docker, alias `podman` to the `docker` binary.


### Kubernetes Cluster

**A Kubernetes Cluster is required.**  The cluster can be localised via Rancher, Docker, Minikube, Kind etc. or remote using cloud based clusters such as Oracle Kubernetes Engine (OKE).  For a localised cluster, [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) is preferred as it can store the Operator image without the need for an external Container Registry.

**Access to the Cluster is required.** Access via `kubectl` to the Kubernetes cluster 


## Example Setup and Workflow

This is an example only of setting up your development environment and a standard workflow.

### Setup

If you are using a different OS/Architecture, please feel encouraged to provide additional instructions.

Note the following images are pulled as part of the development:
* docker.io/kindest/node:v1.29.2 (when using Kind)
* docker.io/moby/buildkit:buildx-stable-1
* container-registry.oracle.com/os/oraclelinux:9-slim

#### MacOS(Intel)
This example was tested on MacOS(Intel) using a Kind cluster and `podman`.

1. Install Software using Brew
    ```bash
    brew install kind podman kubectl golang operator-sdk
    ```
2. Setup the `podman` helper
    ```bash
    PODMAN_VERSION=$(podman -v |awk '{print $NF}')
    sudo /usr/local/Cellar/podman/${PODMAN_VERSION}/bin/podman-mac-helper install
    ```
3. Symlink `docker` to `podman`; this is for `kind load docker-image` to work
    ```bash
    ln -s /usr/local/bin/podman /usr/local/bin/docker
    ```
3. Create a `podman` machine:
    During the `init` you may want to increase the cpu/memory/disk (i.e --memory 32768 --cpus 6 --disk-size 1000)

    ```bash
    podman machine init 
    podman machine set --rootful
    podman machine start
    ```

4. Create a Kind cluster:
    ```bash
    kind create cluster --name ords-operator
    ```
5. Verify cluster access:
    ```bash
    kubectl cluster-info --context kind-ords-operator
    ```

#### Linux - RHEL Compatible (x86)

1. Install Software using dnf

    ```bash
    dnf install podman
    ```

2. Install Kind and Kubectl:

    ```bash
    [ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-amd64
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chomd u+x kind kubectl
    ```

4. Create a Kind cluster:
    ```bash
    kind create cluster --name ords-operator
    ```
    
5. Verify cluster access:
    ```bash
    kubectl cluster-info --context kind-ords-operator
    ```


### Workflow

```bash
make generate
make manifests
make docker-build
make kind-load
```

```bash
make undeploy
make deploy
```
