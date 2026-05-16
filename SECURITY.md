# Security and Verification

## Verifying release checksums

```bash
sha256sum -c SHA256SUMS
```

## Verifying signatures

```bash
cosign verify-blob proxera-agent-linux-amd64 \
  --bundle proxera-agent-linux-amd64.cosign.bundle \
  --certificate-identity-regexp="https://github.com/wenisch-tech/proxera-agent" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## Verifying chart signature

```bash
cosign verify-blob proxera-agent-<version>.tgz \
  --bundle proxera-agent-<version>.tgz.cosign.bundle \
  --certificate-identity-regexp="https://github.com/wenisch-tech/proxera-agent" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```
