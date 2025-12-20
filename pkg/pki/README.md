# PKI Package

Certificate authority and mTLS certificate management for Latis.

## Overview

This package provides:
- **CA management** — Generate or load a certificate authority
- **Certificate generation** — Create certificates signed by the CA
- **SPIFFE-compatible identities** — URIs like `spiffe://latis/unit/abc123`
- **TLS config builders** — Ready-to-use mTLS configurations

## Usage

### Bootstrap (single machine)

```bash
# Terminal 1: Initialize PKI and start unit
latis-unit --init-pki

# Terminal 2: Generate cmdr cert and connect
latis --init-pki
```

This creates `~/.latis/pki/` with:
```
ca.crt      # CA certificate
ca.key      # CA private key
unit.crt    # Unit certificate (server)
unit.key    # Unit private key
cmdr.crt    # Cmdr certificate (client)
cmdr.key    # Cmdr private key
```

### Multi-machine deployment

```bash
# On machine A: Generate CA and cmdr cert
latis-unit --init-pki   # Creates CA + unit cert
latis --init-pki        # Creates cmdr cert

# Copy CA to machine B
scp ~/.latis/pki/ca.crt ~/.latis/pki/ca.key remote:~/.latis/pki/

# On machine B: Generate unit cert
latis-unit --init-pki   # Uses existing CA, creates unit cert
```

### Bring Your Own CA

```bash
latis-unit --ca-cert /path/to/ca.crt --ca-key /path/to/ca.key --init-pki
```

## Certificate Identity

Certificates include SPIFFE-compatible URIs in the Subject Alternative Name:

| Component | Identity URI |
|-----------|--------------|
| cmdr | `spiffe://latis/cmdr` |
| unit | `spiffe://latis/unit/<id>` |

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
identity := pki.UnitIdentity("abc123") // spiffe://latis/unit/abc123
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
