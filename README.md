# Fornex Webhook solver for Cert Manager

This is an unofficial webhook solver for [Cert Manager](https://cert-manager.io/) and [Fornex](https://fornex.com).

## Usage

1. Deploy the webhook:

    ```
    helm install fornex-webhook ./deploy/fornex-webhook \
        --set groupName=<your group>
    ```

2. Create a secret containing your [API key](https://fornex.com/my/settings/api/create):

    ```
    kubectl create secret generic fornex-key \
        --from-literal=api-key=<your key> 
    ```

3. Create a role and role binding:

    ```
    kubectl apply -f rbac.yaml
    ```

4. Configure a certificate issuer:

    ```yaml
    apiVersion: cert-manager.io/v1
    kind: ClusterIssuer
    metadata:
      name: letsencrypt
    spec:
      acme:
        server: https://acme-v02.api.letsencrypt.org/directory
        email: <your e-mail>
        privateKeySecretRef:
          name: letsencrypt-key
        solvers:
          - selector:
              dnsZones:
                - <your domain>
            dns01:
              webhook:
                groupName: <your group>
                solverName: fornex
                config:
                  apiKeySecretRef:
                    name: fornex-key
                    key: api-key
    ```

## Running the test suite

All DNS providers **must** run the DNS01 provider conformance testing suite,
else they will have undetermined behaviour when used with cert-manager.

**It is essential that you configure and run the test suite when creating a
DNS01 webhook.**

To run the tests, first put your api key into testdata/fornex-solver/fornex-key.yaml.

Then you can run the test suite with:

```bash
TEST_ZONE_NAME=<your domain>. make test
```
