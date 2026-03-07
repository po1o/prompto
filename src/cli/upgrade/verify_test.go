package upgrade

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerify(t *testing.T) {
	payload := []byte("checksums fixture")
	signature, restore := installTestPublicKey(t)
	defer restore()

	OK := validateSignature(payload, signature(payload))
	assert.True(t, OK)
}

func TestVerifyFail(t *testing.T) {
	payload := []byte("checksums fixture")
	sign, restore := installTestPublicKey(t)
	defer restore()

	invalidSignature := sign([]byte("different payload"))

	OK := validateSignature(payload, invalidSignature)
	assert.False(t, OK)
}

func installTestPublicKey(t *testing.T) (func([]byte) []byte, func()) {
	t.Helper()

	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	encodedPublicKey, err := x509.MarshalPKIXPublicKey(public)
	require.NoError(t, err)

	oldPublicKey := publicKey
	publicKey = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: encodedPublicKey,
	})

	restore := func() {
		publicKey = oldPublicKey
	}

	return func(payload []byte) []byte {
		return ed25519.Sign(private, payload)
	}, restore
}
