# Security and Verification

## Verifying release checksums

```bash
sha256sum -c SHA256SUMS
```

## Verifying signatures

```bash
cosign verify-blob proxera-client-linux-amd64 \
  --bundle proxera-client-linux-amd64.cosign.bundle \
  --certificate-identity-regexp="https://github.com/wenisch-tech/proxera-client" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## Verifying chart signature

```bash
cosign verify-blob proxera-client-<version>.tgz \
  --bundle proxera-client-<version>.tgz.cosign.bundle \
  --certificate-identity-regexp="https://github.com/wenisch-tech/proxera-client" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```
