## TLS

Create a TLS secret, specifying the abosolute path to your certificate and certificate key path.

In the below example, the TLS Secret `mycert` will be created with the contents of `server.crt` and `server.key`

```bash
kubectl create secret tls sa-cert-global --cert=path/to/cert/server.crt --key=path/to/key/server.key
```

Specify the following in `spec.globalSettings` of your manifest, where:
* `secretName:` is the name of the secret
* `cert:` is the file name of the certificate 
* `key:`  is the file name of the certificate key.

```yaml
    certSecret:
      secretName: sa-cert-global
      cert: server.crt
      key: server.key
```