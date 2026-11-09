// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestGenerateSSHKeyPEM_allKinds(t *testing.T) {
	for _, kind := range []string{"ed25519", "rsa2048", "rsa3072", "rsa4096"} {
		pemStr, err := generateSSHKeyPEM(kind)
		if err != nil {
			t.Fatalf("%s: %s", kind, err)
		}
		block, _ := pem.Decode([]byte(pemStr))
		if block == nil || block.Type != "PRIVATE KEY" {
			t.Fatalf("%s: expected PKCS8 PRIVATE KEY PEM, got %q", kind, pemStr[:30])
		}
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			t.Fatalf("%s: parse: %s", kind, err)
		}
		if kind == "ed25519" {
			if _, ok := key.(interface{ Equal(crypto.PrivateKey) bool }); !ok {
				t.Fatalf("key does not support Equal")
			}
			if _, isRSA := key.(*rsa.PrivateKey); isRSA {
				t.Fatalf("ed25519 requested but RSA key returned")
			}
		} else {
			if _, isRSA := key.(*rsa.PrivateKey); !isRSA {
				t.Fatalf("%s requested but non-RSA key returned", kind)
			}
		}
	}
}

func TestGenerateSSHKeyPEM_rejectsUnknownKind(t *testing.T) {
	if _, err := generateSSHKeyPEM("dsa"); err == nil {
		t.Fatal("expected error for unsupported key type")
	}
}
