# Local mTLS Guide

## 1. Overview

Generate the certificates and keys required to establish mutual TLS (mTLS) between a NATS server and a client.

### Certificates Required

| File | Purpose |
|---|---|
| `ca.key` | CA private key used to sign certificates |
| `ca.crt` | CA certificate trusted by NATS and clients |
| `server.key` | NATS server private key |
| `server.crt` | NATS server certificate |
| `client.key` | Client private key |
| `client.crt` | Client certificate presented to NATS |

### Certificate Identities

| Component | Identity |
|---|---|
| Certificate Authority | Local NATS CA |
| NATS Server | localhost |
| Client | payments-api |
| NATS User | payments-api |

The client certificate will contain `DNS:payments-api` as its Subject Alternative Name (SAN). NATS will use this identity to map the certificate to the configured `payments-api` NATS User.

---

## 2. Directory Structure

Create the following directory structure:

```text
nats-mtls-lab/
│
├── certs/
│   ├── ca/
│   │   ├── ca.key
│   │   └── ca.crt
│   │
│   ├── server/
│   │   ├── server.key
│   │   └── server.crt
│   │
│   └── client/
│       ├── client.key
│       └── client.crt
│
└── openssl/
    ├── server.cnf
    └── client.cnf
```

Create the directories:

```bash
mkdir -p nats-mtls-lab/certs/{ca,server,client}
mkdir -p nats-mtls-lab/openssl

cd nats-mtls-lab
```

---

## 3. Generate Certificate Authority (CA)

The CA is the trusted authority responsible for signing both server and client certificates.

### Generate CA Private Key

```bash
openssl genpkey \
  -algorithm RSA \
  -pkeyopt rsa_keygen_bits:2048 \
  -out certs/ca/ca.key
```

### Generate Self-Signed CA Certificate

```bash
openssl req -x509 -new \
  -key certs/ca/ca.key \
  -sha256 \
  -days 3650 \
  -out certs/ca/ca.crt \
  -subj "/CN=Local NATS CA"
```

Generated files:

- `ca.key` - CA private key. Must remain secure.
- `ca.crt` - CA certificate used to validate certificates signed by this CA.

---

## 4. Generate NATS Server Certificate

The NATS server certificate is used by clients to verify the server's identity.

### Step 1: Create Server OpenSSL Configuration

Create `openssl/server.cnf`:

```ini
[req]
prompt = no
distinguished_name = dn
req_extensions = req_ext

[dn]
CN = localhost

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1

[server_ext]
basicConstraints = CA:FALSE
keyUsage = digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
```

### Step 2: Generate Server Private Key

```bash
openssl genpkey \
  -algorithm RSA \
  -pkeyopt rsa_keygen_bits:2048 \
  -out certs/server/server.key
```

### Step 3: Generate Certificate Signing Request (CSR)

```bash
openssl req -new \
  -key certs/server/server.key \
  -out certs/server/server.csr \
  -config openssl/server.cnf
```

### Step 4: Sign Server Certificate Using CA

```bash
openssl x509 -req \
  -in certs/server/server.csr \
  -CA certs/ca/ca.crt \
  -CAkey certs/ca/ca.key \
  -CAcreateserial \
  -out certs/server/server.crt \
  -days 365 \
  -sha256 \
  -extfile openssl/server.cnf \
  -extensions server_ext
```

The resulting server certificate contains:

- DNS SAN: `localhost`
- IP SAN: `127.0.0.1`
- Extended Key Usage: `serverAuth`

These identities allow clients to verify the server when connecting through `localhost` or `127.0.0.1`.

---

## 5. Generate Client Certificate

The client certificate is presented to NATS during the TLS handshake and is used to authenticate the client.

### Step 1: Create Client OpenSSL Configuration

Create `openssl/client.cnf`:

```ini
[req]
prompt = no
distinguished_name = dn
req_extensions = req_ext

[dn]
CN = payments-api

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = payments-api

[client_ext]
basicConstraints = CA:FALSE
keyUsage = digitalSignature,keyEncipherment
extendedKeyUsage = clientAuth
subjectAltName = @alt_names
```

### Step 2: Generate Client Private Key

```bash
openssl genpkey \
  -algorithm RSA \
  -pkeyopt rsa_keygen_bits:2048 \
  -out certs/client/client.key
```

### Step 3: Generate Certificate Signing Request (CSR)

```bash
openssl req -new \
  -key certs/client/client.key \
  -out certs/client/client.csr \
  -config openssl/client.cnf
```

### Step 4: Sign Client Certificate Using CA

```bash
openssl x509 -req \
  -in certs/client/client.csr \
  -CA certs/ca/ca.crt \
  -CAkey certs/ca/ca.key \
  -CAcreateserial \
  -out certs/client/client.crt \
  -days 365 \
  -sha256 \
  -extfile openssl/client.cnf \
  -extensions client_ext
```

The resulting client certificate contains:

- DNS SAN: `payments-api`
- Extended Key Usage: `clientAuth`

**Important:** The `DNS:payments-api` identity is used by NATS to map the client certificate to the configured `payments-api` NATS User.

---

## 6. Verify Generated Certificates

### Verify Server Certificate

```bash
openssl verify \
  -CAfile certs/ca/ca.crt \
  certs/server/server.crt
```

### Verify Client Certificate

```bash
openssl verify \
  -CAfile certs/ca/ca.crt \
  certs/client/client.crt
```

Expected output:



---

## 10. Configure mTLS in NATS Server

Configure NATS to establish TLS connections, validate client certificates and map certificate identities to configured NATS Users.

### Step 1: Create NATS Configuration

Create `nats.conf` in the project root:

```conf
# NATS Client Listener
port: 4222

# TLS / mTLS Configuration
tls {
    cert_file: "./certs/server/server.crt"
    key_file:  "./certs/server/server.key"
    ca_file:   "./certs/ca/ca.crt"

    verify_and_map: true
}

# User Authentication and Authorization
authorization {
    users: [
        {
            user: "payments-api"

            permissions: {
                publish: {
                    allow: ["payments.>"]
                }

                subscribe: {
                    allow: ["payments.>", "_INBOX.>"]
                }
            }
        }
    ]
}
```

### Step 2: Understand the Configuration

| Configuration | Purpose |
|---|---|
| `port` | NATS client connection port |
| `cert_file` | NATS server certificate |
| `key_file` | NATS server private key |
| `ca_file` | CA certificate used to validate client certificates |
| `verify_and_map` | Validates client certificates and maps certificate identity to a configured NATS User |
| `authorization.users` | Defines NATS Users |
| `permissions.publish` | Defines subjects the user can publish to |
| `permissions.subscribe` | Defines subjects the user can subscribe to |

### Step 3: Certificate-to-User Mapping

The client certificate contains:

```text
DNS:payments-api
```

NATS uses this certificate identity to find the configured user:

```conf
user: "payments-api"
```

The mapping flow is:

```text
Client Certificate
       |
       | DNS SAN: payments-api
       v
NATS Certificate Validation
       |
       v
NATS User: payments-api
       |
       v
Subject-Level Authorization
```

**Important:**

- `verify_and_map: true` requires and validates client certificates and enables certificate-to-user mapping.
- The certificate identity must match a configured NATS User.
- A valid certificate alone does not automatically create a NATS User.
- No username/password is required for this authentication flow.
- This example uses the default global Account (`$G`) to keep the initial test simple.

### Step 4: Start NATS Server

Validate the configuration:

```bash
nats-server -t -c nats.conf
```

Start NATS:

```bash
nats-server -c nats.conf
```

NATS should start successfully and listen on port `4222`.

---

## 11. Use mTLS Certificates in Client Code

Once NATS is configured, clients must present their certificate and private key while establishing a connection.

The client also needs the CA certificate to validate the NATS server certificate.

### Client Configuration Requirements

| File | Client Usage |
|---|---|
| `ca.crt` | Validate NATS server certificate |
| `client.crt` | Authenticate client with NATS |
| `client.key` | Prove possession of client private key |

The client does not need:
- `ca.key`
- `server.key`
- `server.crt`

### Option 1: NATS CLI

The NATS CLI can be used to validate mTLS before introducing application code.

**Publish a message:**

```bash
nats pub payments.created "Hello from mTLS client" \
  --server tls://localhost:4222 \
  --tlsca certs/ca/ca.crt \
  --tlscert certs/client/client.crt \
  --tlskey certs/client/client.key
```

**Subscribe to messages:**

```bash
nats sub "payments.>" \
  --server tls://localhost:4222 \
  --tlsca certs/ca/ca.crt \
  --tlscert certs/client/client.crt \
  --tlskey certs/client/client.key
```

The CLI establishes a TLS connection using the supplied certificates.

### Option 2: Go Application

Install the NATS Go client:

```bash
go get github.com/nats-io/nats.go
```

Create `main.go`:

```go
package main

import (
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "os"

    "github.com/nats-io/nats.go"
)

func main() {

    // Load trusted CA certificate
    caCert, err := os.ReadFile("certs/ca/ca.crt")
    if err != nil {
        panic(err)
    }

    caPool := x509.NewCertPool()

    if !caPool.AppendCertsFromPEM(caCert) {
        panic("Failed to load CA certificate")
    }

    // Load client certificate and private key
    clientCert, err := tls.LoadX509KeyPair(
        "certs/client/client.crt",
        "certs/client/client.key",
    )
    if err != nil {
        panic(err)
    }

    // Configure TLS
    tlsConfig := &tls.Config{
        RootCAs:      caPool,
        Certificates: []tls.Certificate{clientCert},
        MinVersion:   tls.VersionTLS12,
    }

    // Establish NATS connection
    nc, err := nats.Connect(
        "tls://localhost:4222",
        nats.Secure(tlsConfig),
    )

    if err != nil {
        panic(err)
    }

    defer nc.Close()

    fmt.Println("Connected to NATS using mTLS")

    // Publish message
    err = nc.Publish(
        "payments.created",
        []byte("Hello from Go mTLS client"),
    )

    if err != nil {
        panic(err)
    }

    fmt.Println("Message published successfully")
}
```

Run the application:

```bash
go run main.go
```

Expected output:

```text
Connected to NATS using mTLS
Message published successfully
```

### Understanding Client TLS Configuration

| Go Configuration | Purpose |
|---|---|
| `RootCAs` | CA certificates used to validate the NATS server |
| `Certificates` | Client certificate and corresponding private key |
| `MinVersion` | Minimum permitted TLS protocol version |
| `nats.Secure()` | Configures the NATS connection to use the supplied TLS configuration |

The Go application performs two important operations:

1. Validates the NATS server certificate using the trusted CA.
2. Presents its client certificate and proves possession of its private key during the TLS handshake.

NATS then validates the client certificate and maps its identity to the configured NATS User.

---

## 12. Authentication and Authorization Validation

After successfully connecting, validate both authentication and authorization independently.

| Test Scenario | Expected Result |
|---|---|
| Valid CA and valid client certificate | Connection succeeds |
| Client certificate not provided | TLS handshake rejected |
| Client certificate signed by an untrusted CA | TLS handshake rejected |
| Valid certificate with no matching configured NATS User | Authentication rejected |
| Valid user publishes to `payments.created` | Publish succeeds |
| Valid user publishes to `orders.created` | Authorization denied |

**Example: Test Authorization Failure**

```bash
nats pub orders.created "Should be rejected" \
  --server tls://localhost:4222 \
  --tlsca certs/ca/ca.crt \
  --tlscert certs/client/client.crt \
  --tlskey certs/client/client.key
```

The TLS connection should authenticate successfully, but the publish should be rejected because the `payments-api` user does not have permission to publish to `orders.created`.

**Final Expected Flow:**

```text
Application
    |
    | Client Certificate + Private Key
    v
NATS TLS Handshake
    |
    | Validate Certificate Against CA
    v
Certificate Identity Mapping
    |
    | DNS SAN: payments-api
    v
NATS User: payments-api
    |
    | Evaluate Subject Permissions
    v
Publish / Subscribe Access
```
