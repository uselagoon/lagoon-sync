package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

/**
* generatePrivateKey is used to generate a random private key - we're using this in our tests
 */
func generatePrivateKey(outputDir string) (string, error) {
	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	// Encode private key to PEM format
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes})

	// Generate a random filename
	randomFilename := fmt.Sprintf("private_key_%d.pem", time.Now().UnixNano())

	// Save private key to a file in the specified directory with the random filename
	privateKeyPath := filepath.Join(outputDir, randomFilename)
	err = os.WriteFile(privateKeyPath, privateKeyPEM, 0600)
	if err != nil {
		return "", err
	}

	return privateKeyPath, nil
}

const test_findSSHKeyFilesNumber = 3

func Test_findSSHKeyFiles(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "Run on test directory",
			want:    test_findSSHKeyFilesNumber,
			wantErr: false,
		},
	}

	// Let's generate the files
	tmpDir, err := os.MkdirTemp("", "keypair_test")
	if err != nil {
		t.Fatal("Unable to create temporary directory: ", err.Error())
	}

	defer func() {
		os.RemoveAll(tmpDir)
	}()

	privateKeys := []string{}

	for i := 0; i < test_findSSHKeyFilesNumber; i++ {
		key, err := generatePrivateKey(tmpDir)
		if err != nil {
			t.Fatal("Unable to create private key: ", err.Error())
		}
		privateKeys = append(privateKeys, key)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSSHAuthMethodsFromDirectory(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSSHAuthMethodsFromDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("getSSHAuthMethodsFromDirectory() got = %v, want %v", got, tt.want)
			}
		})
	}
}
