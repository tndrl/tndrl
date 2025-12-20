package pki

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCA(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	if ca.Cert == nil {
		t.Error("CA certificate is nil")
	}
	if ca.Key == nil {
		t.Error("CA private key is nil")
	}

	// Verify CA properties
	if !ca.Cert.IsCA {
		t.Error("Certificate is not marked as CA")
	}
	if ca.Cert.Subject.CommonName != "Latis CA" {
		t.Errorf("CommonName = %q, want %q", ca.Cert.Subject.CommonName, "Latis CA")
	}
	if ca.Cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("CA missing KeyUsageCertSign")
	}
}

func TestCAExists(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		if CAExists(dir) {
			t.Error("CAExists() = true for empty directory, want false")
		}
	})

	t.Run("with CA files", func(t *testing.T) {
		dir := t.TempDir()
		ca, err := GenerateCA()
		if err != nil {
			t.Fatalf("GenerateCA() error = %v", err)
		}
		if err := ca.Save(dir); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if !CAExists(dir) {
			t.Error("CAExists() = false after saving CA, want true")
		}
	})
}

func TestSaveAndLoadCA(t *testing.T) {
	dir := t.TempDir()

	// Generate and save
	original, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}
	if err := original.Save(dir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load and compare
	loaded, err := LoadCAFromDir(dir)
	if err != nil {
		t.Fatalf("LoadCAFromDir() error = %v", err)
	}

	// Compare certificates
	if !original.Cert.Equal(loaded.Cert) {
		t.Error("Loaded certificate does not match original")
	}

	// Compare keys by comparing public key parameters
	if !original.Key.PublicKey.Equal(&loaded.Key.PublicKey) {
		t.Error("Loaded key public key does not match original")
	}
	// Compare private key D parameter
	if original.Key.D.Cmp(loaded.Key.D) != 0 {
		t.Error("Loaded key private component does not match original")
	}
}

func TestLoadCA_InvalidFiles(t *testing.T) {
	t.Run("missing files", func(t *testing.T) {
		dir := t.TempDir()
		_, err := LoadCAFromDir(dir)
		if err == nil {
			t.Error("LoadCAFromDir() expected error for missing files")
		}
	})

	t.Run("invalid cert path", func(t *testing.T) {
		_, err := LoadCA("/nonexistent/ca.crt", "/nonexistent/ca.key")
		if err == nil {
			t.Error("LoadCA() expected error for nonexistent paths")
		}
	})

	t.Run("corrupt cert file", func(t *testing.T) {
		dir := t.TempDir()
		certPath := filepath.Join(dir, "ca.crt")
		keyPath := filepath.Join(dir, "ca.key")

		// Write garbage
		if err := writeTestFile(certPath, "not a certificate"); err != nil {
			t.Fatal(err)
		}
		if err := writeTestFile(keyPath, "not a key"); err != nil {
			t.Fatal(err)
		}

		_, err := LoadCA(certPath, keyPath)
		if err == nil {
			t.Error("LoadCA() expected error for corrupt files")
		}
	})
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
