# Example

This example walks through using the **ORDS Operator** with a Containerised Oracle Database created by the **OraOperator** in the same Kubernetes Cluster.

### Install Cert-Manager

The OraOperator uses webhooks for validating user input before persisting it in etcd. 
Webhooks require TLS certificates that are generated and managed by a certificate manager.

Install [cert-manager](https://cert-manager.io/) with the following command:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.5/cert-manager.yaml
```
<sup>latest cert-manager version, **v1.14.5**, valid as of **2-May-2024**</sup>

Check that all pods have a STATUS of **Running** before proceeding to the next step:
```bash
kubectl -n cert-manager get pods
```

Review [cert-managers installation documentation](https://cert-manager.io/docs/installation/kubectl/) for more information.

### Install OraOperator

Install the [Oracle Operator for Kubernetes](https://github.com/oracle/oracle-database-operator/tree/main):

```bash
kubectl apply -f https://raw.githubusercontent.com/oracle/oracle-database-operator/main/oracle-database-operator.yaml
```

### Install ORDS Operator

Install the Oracle ORDS Operator:

```bash
kubectl apply -f https://github.com/gotsysdba/oracle-ords-operator/releases/latest/download/oracle-ords-operator.yaml
```

### Bind the OraOperator to the ADB

1. Obtain the OCID of the ADB and set to an environment variable:

  ```
  export ADB_OCID=<insert OCID here>
  ```

1. Create a manifest to bind to the ADB.

    ```bash
    echo "
    apiVersion: database.oracle.com/v1alpha1
    kind: AutonomousDatabase
    metadata:
      name: adb-existing
    spec:
      hardLink: false
      details:
        autonomousDatabaseOCID: $ADB_OCID
        wallet:
          name: adb-tns-admin
          password:
            k8sSecret:
              name: db-auth" | kubectl apply -f -
    ```

1. Apply the Container Oracle Database manifest:
    ```bash
    kubectl apply -f ordspoc-sidb.yaml
    ```

1. Watch the `singleinstancedatabases` resource until the database status is **Healthy**:

    ```bash
    kubectl get singleinstancedatabases/ordspoc-sidb -w
    ```

    **NOTE**: If this is the first time pulling the free database image, it may take up to 15 minutes for the database to become available.

### Create RestDataServices Resource

1. Retrieve the Connection String from the containerised SIDB.

    ```bash
    CONN_STRING=$(kubectl get singleinstancedatabase ordspoc-sidb \
      -o jsonpath='{.status.pdbConnectString}')

    echo $CONN_STRING
    ```

1. Create a manifest for ORDS.

    As the DB in the Free image does not contain ORDS (or APEX), the following additional keys are specified for the pool:
    * `autoUpgradeORDS` - Boolean; when true the ORDS will be installed/upgraded in the database
    * `autoUpgradeAPEX` - Boolean; when true the APEX will be installed/upgraded in the database
    * `db.adminUser` - User with privileges to install, upgrade or uninstall ORDS in the database (SYS).
    * `db.adminUser.secret` - Secret containing the password for `db.adminUser` (created in the first step)

    The `db.username` will be used as the ORDS schema in the database during the install/upgrade process (ORDS_PUBLIC_USER).

    ```bash
    echo "
    apiVersion: database.oracle.com/v1
    kind: RestDataServices
    metadata:
      name: ordspoc-server
    spec:
      image: container-registry.oracle.com/database/ords:24.1.0
      forceRestart: true
      globalSettings:
        database.api.enabled: true
      poolSettings:
        - poolName: ORDSPOC
          autoUpgradeORDS: true
          autoUpgradeAPEX: true
          restEnabledSql.active: true
          plsql.gateway.mode: direct
          db.connectionType: customurl
          db.customURL: jdbc:oracle:thin:@//${CONN_STRING}
          db.secret:
            secretName:  db-auth
            passwordKey: password
          db.adminUser: SYS
          db.adminUser.secret:
            secretName:  db-auth
            passwordKey: password" | kubectl apply -f -
    ```
    <sup>latest container-registry.oracle.com/database/ords version, **24.1.0**, valid as of **2-May-2024**</sup>

1. Apply the Container Oracle Database manifest:
    ```bash
    kubectl apply -f ordspoc-server.yaml
    ```

1. Watch the restdataservices resource until the status is **Healthy**:
    ```bash
    kubectl get restdataservices ordspoc-server -w
    ```

    **NOTE**: If this is the first time pulling the ORDS image, it may take up to 5 minutes.  If APEX
    is being installed for the first time by the Operator, it may remain in the **Preparing** 
    status for an additional 5 minutes.

### Test

Open a port-forward to the ORDS service, for example:

```bash
kubectl port-forward service/ordspoc-server 8443:8443
```

Direct your browser to: `https://localhost:8443/ords/ordspoc`
