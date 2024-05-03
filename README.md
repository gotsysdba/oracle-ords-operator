# Oracle Rest Data Services (ORDS) Operator for Kubernetes

This is a **Proof-of-Concept** Oracle Rest Data Services Operator (ORDS Operator) and is *currently* **not supported** by Oracle.

## Description

The ORDS Operator extends the Kubernetes API with custom resources and controllers for automating Oracle Rest Data
Services lifecycle management.  Using the ORDS Operator, you can easily migrate existing, or create new, ORDS implementations
into an existing Kubernetes cluster.

## Features Summary

The custom RestDataServices resource supports the following configurations as either a Deployment, StatefulSet, or DaemonSet:

* Single RestDataServices resource with one database pool
* Single RestDataServices resource with multiple database pools
* Multiple RestDataServices resources, each with one database pool
* Multiple RestDataServices resources, each with multiple database pools

It supports the majority of ORDS configuration settings as per the [API Documentation](docs/api.md)

The ORDS and APEX schemas can be automatically installed/upgraded into the database by the ORDS Operator.

### Prerequisites
- go version v1.20.0+
- docker/podman version 17.03+. (for podman, make sure the docker command is aliased to podman)
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/oracle-ords-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.  And it is required to have access to pull the image from the working environment.  Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/oracle-ords-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## Contributing
See [Contributing to this Repository](./CONTRIBUTING.md)

## Reporting a Security Issue

See [Reporting security vulnerabilities](./SECURITY.md)

## License

Copyright (c) 2024 Oracle and/or its affiliates.
Released under the Universal Permissive License v1.0 as shown at [https://oss.oracle.com/licenses/upl/](https://oss.oracle.com/licenses/upl/)

