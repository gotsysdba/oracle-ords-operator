## Connectivy

### Username/Password Secret

```bash
kubectl create secret generic db-user-pass \
    --from-literal=password='S!B\*d$zDsb='
```

### TNS_ADMIN ConfigMap (Optional)