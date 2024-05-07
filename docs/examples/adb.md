# Example

This example walks through using the **ORDS Operator** with an Oracle Autonomous Database.  

This assumes that an ADB has already been provisioned and is configured as "Secure Access from Anywhere".  
Note that if behind a Proxy, this example will not work as the Wallet will need to be modified to support the proxy configuration.

### Install ORDS Operator

Install the Oracle ORDS Operator:

```bash
kubectl apply -f https://github.com/gotsysdba/oracle-ords-operator/releases/latest/download/oracle-ords-operator.yaml
```

### ADB Wallet Secret

Create a Secret with the wallet for the ADB, replacing `<full_path_to_wallet.zip>` with the path to the wallet zip file:

```bash
kubectl create secret generic adb-wallet \
  --from-file=<full_path_to_wallet.zip>
```

For example:

```bash
kubectl create secret generic adb-wallet \
  --from-file=~/Downloads/Wallet_ADBPOC.zip
```

### ADB ADMIN Password Secret

Create a Secret for the ADB Admin password, replacing `<admin_password>` with the real ADMIN password:

```bash
kubectl create secret generic db-auth \
  --from-literal=password=<admin_password>
```

For example:
```bash
kubectl create secret generic db-auth \
  --from-literal=password=horse-battery-staple
```

### Create RestDataServices Resource

1. Create a manifest for ORDS.

    As an ADB already maintains ORDS and APEX, `autoUpgradeORDS` and `autoUpgradeAPEX` will be ignored if set.  A new DB User for ORDS will be created to avoid conflict with the pre-provisioned one.  This user will be
    named, `ORDS_PUBLIC_USER_OPER` if `db.username` is either not specified or set to `ORDS_PUBLIC_USER`.

    ```bash
    echo "
    apiVersion: database.oracle.com/v1
    kind: RestDataServices
    metadata:
      name: ordspoc-server
    spec:
      image: container-registry.oracle.com/database/ords:23.4.0
      forceRestart: true
      globalSettings:
        database.api.enabled: true
      poolSettings:
        - poolName: ORDSPOC
          db.wallet.zip.service: adbpoc_tp
          dbWalletSecret:
            secretName:  adb-wallet
            walletName: Wallet_ADBPOC.zip
          restEnabledSql.active: true
          feature.sdw: true
          plsql.gateway.mode: proxied
          db.username: ORDS_PUBLIC_USER_OPER
          db.secret:
            secretName:  db-auth
            passwordKey: password
          db.adminUser: ADMIN
          db.adminUser.secret:
            secretName:  db-auth
            passwordKey: password" | kubectl apply -f -
    ```
    <sup>24.1.0 cannot be used due to a image ENV issue</sup>

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
