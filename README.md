# Oracle Rest Data Services (ORDS) Operator for Kubernetes

This is a **Proof-of-Concept** Oracle Rest Data Services Operator (ORDS Operator) and is *currently* **not supported** by Oracle.

Your evaluation and [Feedback](/issues/new?labels=feedback&title=New+Feedback) is appreciated.

## Description

The ORDS Operator extends the Kubernetes API with custom resources and controllers for automating Oracle Rest Data
Services lifecycle management.  Using the ORDS Operator, you can easily migrate existing, or create new, ORDS implementations
into an existing Kubernetes cluster.

## Features Summary

The custom RestDataServices resource supports the following configurations as either a Deployment, StatefulSet, or DaemonSet:

* Single RestDataServices resource with one database pool
* Single RestDataServices resource with multiple database pools<sup>*</sup>
* Multiple RestDataServices resources, each with one database pool
* Multiple RestDataServices resources, each with multiple database pools<sup>*</sup>

It supports the majority of ORDS configuration settings as per the [API Documentation](docs/api.md)

The ORDS and APEX schemas can be [automatically installed/upgraded](docs/autoupgrade.md) into the Oracle Database by the ORDS Operator.

<sup>*See [Limitations](#limitations)</sup>

### Quick Installation

To install the ORDS Operator, run:

```bash
kubectl apply -f https://github.com/gotsysdba/oracle-ords-operator/releases/latest/download/oracle-ords-operator.yaml
```

This will create a new namespace, `oracle-ords-operator-system`, in which the Controller will run.

### Common Configurations

A few common configuration examples can be used to quickly familiarise yourself with the ORDS Custom Resource Definition.
The "Conclusion" section of each example highlights specific settings to enable functionality that maybe of interest.

* [Containerised Single Instance Database using the OraOperator](docs/examples/sidb_container.md)
* [Autonomous Database using the OraOperator](docs/examples/adb_oraoper.md) <sup>*See [Limitations](#limitations)</sup>
* [Autonomous Database without the OraOperator](docs/examples/adb.md)
* [Multipool, Multidatabase using a TNS Names file](docs/examples/multi_pool.md)

Running through all examples in the same Kubernetes cluster illustrates the ability to run multiple ORDS instances with a variety of different configurations.

If you have a specific use-case that is not covered and would like it to be, please open an [Enhancement Request](issues/new?labels=enhancement) or feel free to contribute it via a Pull Request.

### Limitations

When connecting to a mTLS enabled ADB and using the OraOperator to retreive the Wallet, it is currently not supported to have multiple, different databases supported by the single RestDataServices resource.  This is due to a requirement to set the `TNS_ADMIN` parameter at the Pod level ([#97](https://github.com/oracle/oracle-database-operator/issues/97)).

## Contributing
See [Contributing to this Repository](./CONTRIBUTING.md)

## Reporting a Security Issue

See [Reporting security vulnerabilities](./SECURITY.md)

## License

Copyright (c) 2024 Oracle and/or its affiliates.
Released under the Universal Permissive License v1.0 as shown at [https://oss.oracle.com/licenses/upl/](https://oss.oracle.com/licenses/upl/)
