# Authentication

- Core Principle - Authentication answers "Who is connecting to NATS?"
- Identity Verification - NATS validates client identity before establishing a connection.
- Separation of Concerns - Authentication establishes identity; Authorization (SBAC) governs what that identity can do.

---

## 1. Authentication Mechanisms

| Mechanism | Credential | Operational Fit | Trade-offs |
|---|---|---|---|
| Username / Password | Plaintext string / bcrypt | Local development, baseline evaluation | Simple; requires credential rotation and secret distribution. |
| Token | Shared secret string | Simple scripts, webhooks | Shared identity; lacks granular per-client auditing. |
| TLS / mTLS | X.509 Client Certificates | Enterprise production (Platform standard) | Strong cryptographic identity; relies on enterprise PKI and certificate lifecycle. |
| NKey | Public-key seed (Ed25519) | Modern zero-trust microservices | Fast challenge-response; private key never traverses network. |
| JWT + NKey | Decentralized signed claims | Multi-tenant Operator model | Self-service distributed administration; requires tooling (`nsc`). |
| Auth Callout | External webhook / gRPC | Corporate IAM / OAuth integration | Centralized directory sync; introduces external network dependency. |

---

## 2. Mutual TLS (mTLS) Authentication

Enterprise standard mechanism combining transport security with cryptographic identity verification.

### Configuration (`verify_and_map`)

```conf
tls {
  cert_file: "/etc/nats/certs/server-cert.pem"
  key_file:  "/etc/nats/certs/server-key.pem"
  ca_file:   "/etc/nats/certs/ca.pem"
  verify_and_map: true
}
```

- `verify_and_map: true` - Enforces client certificate validation and automatically maps the certificate identity to a configured NATS User. Replaces standalone `verify: true`.
- Trusted CA - Client certificates must chain to `ca_file`. Untrusted or expired certificates are terminated during the TLS handshake.

### Certificate Identity Mapping

NATS extracts the client identity from the X.509 certificate in order of precedence:

1. Subject Alternative Name (SAN) - Checks SAN DNS names and URI/email attributes first (e.g. `payments-api.service.internal`).
2. Distinguished Name (DN) - Falls back to the Common Name (CN) or full DN (e.g. `CN=payments-api,OU=Engineering`).
3. User Resolution - The extracted identity must exactly match a declared `user` in the NATS account configuration. A valid certificate without a matching NATS user is rejected.

---

## 3. Operational Requirements

- L4 Proxy Ingress - Reverse proxies (e.g. HAProxy) must run in TCP passthrough mode so NATS receives the raw TLS handshake and client certificate.
- Identity Conventions - Standardize SAN naming (e.g. `<service>.<env>.nats.internal`) across the enterprise PKI to prevent mapping ambiguity.
- Secret Hygiene - Never commit private keys or seed keys to source control; distribute credentials via Kubernetes Secrets or enterprise vault.
