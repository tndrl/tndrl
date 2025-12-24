# PKI Package

Certificate authority and mTLS certificate management for Latis.

## Overview

This package provides:
- **CA management** — Generate or load a certificate authority
- **Certificate generation** — Create certificates signed by the CA
- **SPIFFE-compatible identities** — URIs like `spiffe://latis/node/abc123`
- **TLS config builders** — Ready-to-use mTLS configurations

## Usage

### Initialize PKI

```bash
# Start a node with PKI initialization
latis serve -c config.yaml --pki-init

# Or use a config file with pki.init: true
```

This creates `~/.latis/pki/` with:
```
ca.crt      # CA certificate
ca.key      # CA private key (only on first node)
latis.crt   # Node certificate
latis.key   # Node private key
```

### Multi-Machine Deployment

```bash
# On machine A: Generate CA and node cert
latis serve -c config.yaml --pki-init

# Copy CA to machine B (but NOT ca.key unless needed)
scp ~/.latis/pki/ca.crt remote:~/.latis/pki/
scp ~/.latis/pki/ca.key remote:~/.latis/pki/   # Optional: only if B needs to issue certs

# On machine B: Generate node cert using existing CA
latis serve -c config.yaml --pki-init
```

### Bring Your Own CA

```bash
latis serve --pki-ca-cert /path/to/ca.crt --pki-ca-key /path/to/ca.key --pki-init
```

Or in config:
```yaml
pki:
  dir: ~/.latis/pki
  caCert: /path/to/ca.crt
  caKey: /path/to/ca.key
  init: true
```

## Certificate Identity

Certificates include SPIFFE-compatible URIs in the Subject Alternative Name:

| Usage | Identity URI |
|-------|--------------|
| Node | `spiffe://latis/node/<name>` |

This enables future integration with SPIFFE/SPIRE for automatic certificate management.

## API

### CA Operations

```go
// Generate new CA
ca, err := pki.GenerateCA()

// Load existing CA
ca, err := pki.LoadCA(certPath, keyPath)

// Save CA to directory
err := ca.Save("/path/to/pki")

// Check if CA exists
exists := pki.CAExists("/path/to/pki")
```

### Certificate Operations

```go
// Generate certificate signed by CA
identity := pki.NodeIdentity("my-agent") // spiffe://latis/node/my-agent
cert, err := pki.GenerateCert(ca, identity, isServer, isClient)

// Load existing certificate
cert, err := pki.LoadCert(certPath, keyPath)

// Save certificate
err := cert.Save(certPath, keyPath)

// Get tls.Certificate for TLS config
tlsCert, err := cert.TLSCertificate()
```

### TLS Config

```go
// Server config (requires client certs)
tlsConfig, err := pki.ServerTLSConfig(serverCert, ca)

// Client config (presents cert, verifies server)
tlsConfig, err := pki.ClientTLSConfig(clientCert, ca, "localhost")
```

## Security

- **ECDSA P-256** keys (fast, secure)
- **TLS 1.3** minimum version
- **mTLS** — mutual authentication required
- CA valid for 10 years, certificates valid for 1 year
- Private keys stored with 0600 permissions
- IP SANs include 127.0.0.1 and ::1 for localhost testing
