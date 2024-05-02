# Example

This example walks through using the **ORDS Operator** with a Containerised Oracle Database created by the **OraOperator** in the same Kubernetes Cluster.

### Install Cert-Manager

The OraOperator uses webhooks for validating user input before persisting it in etcd. 
Webhooks require TLS certificates that are generated and managed by a certificate manager.

Install [cert-manager](https://cert-manager.io/) with the following command:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.5/cert-manager.yaml
```

Review [cert-managers installation documentation](https://cert-manager.io/docs/installation/kubectl/) for more information.

<sup>latest cert-manager version, v1.14.5, valid as of 28-Apr-2024</sup>

### Install OraOperator

Install the [Oracle Operator for Kubernetes](https://github.com/oracle/oracle-database-operator/tree/main)

```bash
kubectl apply -f https://raw.githubusercontent.com/oracle/oracle-database-operator/main/oracle-database-operator.yaml
```

### Install ORDS Operator

<TODO>

### Deploy a Containerised Oracle Database

1. Create a Secret for the Database password:

    ```bash
    DB_PWD=$(echo "ORDSPOC_$(date +%H%S%M)")

    kubectl create secret generic db-auth \
      --from-literal=password=${DB_PWD}
    ```
1. Create a manifest for the containerised Oracle Database.

    The POC uses an Oracle Free Image, but other versions may be subsituted; review the OraOperator Documentation for details on the manifests.

    ```bash
    cat > ordspoc-sidb.yaml <<- EOF
    apiVersion: database.oracle.com/v1alpha1
    kind: SingleInstanceDatabase
    metadata:
      name: ordspoc-sidb
    spec:
      replicas: 1
      image:
        pullFrom: container-registry.oracle.com/database/free:23.3.0.0
        prebuiltDB: true
      sid: FREE
      edition: free
      adminPassword:
        secretName: db-auth
        secretKey: password
      pdbName: FREEPDB1
    EOF
    ```

1. Apply the Container Oracle Database manifest:
    ```bash
    kubectl apply -f ordspoc-sidb.yaml
    ```

1. Watch the singleinstancedatabases resource until the database status is **Healthy**:

    ```bash
    kubectl get singleinstancedatabases/ordspoc-sidb -w
    ```

    **NOTE**: If this is the first time pulling the free database image, it may take up to 15 minutes for the database to become available.

### Create RestDataServices Resource

1. Retrieve the Connection String from the SIDB.

    ```bash
    CONN_STRING=$(kubectl get singleinstancedatabase ordspoc-sidb \
      -o jsonpath='{.status.pdbConnectString}')

    echo $CONN_STRING
    ```

1. Create a manifest for ORDS.

    As the DB in the Free image does not contain ORDS, the following additional keys are specified for the pool:
    * `db.adminUser` - User with privileges to install, upgrade or uninstall ORDS in the database (SYS).
    * `db.adminUser.secret` - Secret containing the password for `db.adminUser` (created in the first step)
    * `autoUpgradeORDS` - Boolean; when true the ORDS will be installed/upgraded in the database
    * `autoUpgradeAPEX` - Boolean; when true the APEX will be installed/upgraded in the database

    The `db.username` will be used as the ORDS schema in the database during the install/upgrade process (ORDS_PUBLIC_USER).

    ```bash
    cat > ordspoc-server.yaml <<- EOF
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
          restEnabledSql.active: true
          feature.sdw: true
          plsql.gateway.mode: direct
          db.connectionType: customurl
          db.customURL: jdbc:oracle:thin:@//${CONN_STRING}
          db.username: ORDS_PUBLIC_USER
          db.secret:
            secretName:  db-auth
            passwordKey: password
          db.adminUser: SYS
          db.adminUser.secret:
            secretName:  db-auth
            passwordKey: password
    EOF
    ```
    **NOTE**: If this is the first time pulling the ORDS image, it may take up to 5 minutes.


1. Apply the Container Oracle Database manifest:
    ```bash
    kubectl apply -f ordspoc-server.yaml
    ```

1. Watch the singleinstancedatabases resource until the database is available:
    ```bash
    kubectl get restdataservices ordspoc-server -w
    ```

### Test

Open a port-forward to the ORDS service, for example:

```bash
kubectl port-forward service/ordspoc-server 8443:8443
```

Direct your browser to: `https://localhost:8443/ords/ordspoc`

## Conclusion

