// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func generateSSHKeyPEM(kind string) (string, error) {
	var key crypto.PrivateKey
	var err error

	switch kind {
	case "ed25519":
		_, key, err = ed25519.GenerateKey(rand.Reader)
	case "rsa2048":
		key, err = rsa.GenerateKey(rand.Reader, 2048)
	case "rsa3072":
		key, err = rsa.GenerateKey(rand.Reader, 3072)
	case "rsa4096":
		key, err = rsa.GenerateKey(rand.Reader, 4096)
	default:
		return "", fmt.Errorf("unsupported key type %q (use ed25519, rsa2048, rsa3072, or rsa4096)", kind)
	}
	if err != nil {
		return "", fmt.Errorf("generating %s key: %w", kind, err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("marshaling %s private key to PKCS8: %w", kind, err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}
