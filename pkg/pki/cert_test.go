package pki

import (
	"crypto/x509"
	"path/filepath"
	"testing"
)

func TestGenerateCert(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	identity := "spiffe://latis/node/test123"
	cert, err := GenerateCert(ca, identity, true, true)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}

	if cert.Cert == nil {
		t.Error("Certificate is nil")
	}
	if cert.Key == nil {
		t.Error("Private key is nil")
	}

	// Verify signed by CA
	roots := x509.NewCertPool()
	roots.AddCert(ca.Cert)
	_, err = cert.Cert.Verify(x509.VerifyOptions{Roots: roots})
	if err != nil {
		t.Errorf("Certificate not verified by CA: %v", err)
	}
}

func TestGenerateCert_ServerOnly(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	cert, err := GenerateCert(ca, "spiffe://latis/node/server", true, false)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}

	hasServerAuth := false
	hasClientAuth := false
	for _, usage := range cert.Cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
		if usage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
	}

	if !hasServerAuth {
		t.Error("Server cert missing ServerAuth usage")
	}
	if hasClientAuth {
		t.Error("Server-only cert should not have ClientAuth usage")
	}
}

func TestGenerateCert_ClientOnly(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	cert, err := GenerateCert(ca, "spiffe://latis/node/client", false, true)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}

	hasServerAuth := false
	hasClientAuth := false
	for _, usage := range cert.Cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
		if usage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
	}

	if hasServerAuth {
		t.Error("Client-only cert should not have ServerAuth usage")
	}
	if !hasClientAuth {
		t.Error("Client cert missing ClientAuth usage")
	}
}

func TestGenerateCert_Both(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	cert, err := GenerateCert(ca, "spiffe://latis/node/both", true, true)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}

	hasServerAuth := false
	hasClientAuth := false
	for _, usage := range cert.Cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
		if usage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
	}

	if !hasServerAuth {
		t.Error("Dual cert missing ServerAuth usage")
	}
	if !hasClientAuth {
		t.Error("Dual cert missing ClientAuth usage")
	}
}

func TestCertIdentity(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	identity := "spiffe://latis/node/myid"
	cert, err := GenerateCert(ca, identity, true, false)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}

	// Check SAN URIs
	found := false
	for _, uri := range cert.Cert.URIs {
		if uri.String() == identity {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Identity URI %q not found in certificate SANs: %v", identity, cert.Cert.URIs)
	}

	// Check CommonName
	if cert.Cert.Subject.CommonName != identity {
		t.Errorf("CommonName = %q, want %q", cert.Cert.Subject.CommonName, identity)
	}
}

func TestSaveAndLoadCert(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "test.crt")
	keyPath := filepath.Join(dir, "test.key")

	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	// Generate and save
	original, err := GenerateCert(ca, "spiffe://latis/test", true, true)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}
	if err := original.Save(certPath, keyPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load and compare
	loaded, err := LoadCert(certPath, keyPath)
	if err != nil {
		t.Fatalf("LoadCert() error = %v", err)
	}

	if !original.Cert.Equal(loaded.Cert) {
		t.Error("Loaded certificate does not match original")
	}
}

func TestCertExists(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "test.crt")
	keyPath := filepath.Join(dir, "test.key")

	if CertExists(certPath, keyPath) {
		t.Error("CertExists() = true for nonexistent files, want false")
	}

	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	cert, err := GenerateCert(ca, "spiffe://latis/test", true, true)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}
	if err := cert.Save(certPath, keyPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !CertExists(certPath, keyPath) {
		t.Error("CertExists() = false after saving cert, want true")
	}
}

func TestTLSCertificate(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	cert, err := GenerateCert(ca, "spiffe://latis/test", true, true)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}

	tlsCert, err := cert.TLSCertificate()
	if err != nil {
		t.Fatalf("TLSCertificate() error = %v", err)
	}

	if len(tlsCert.Certificate) == 0 {
		t.Error("TLS certificate has no certificate data")
	}
	if tlsCert.PrivateKey == nil {
		t.Error("TLS certificate has no private key")
	}
}
