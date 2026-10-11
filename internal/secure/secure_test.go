package secure

import (
	"bytes"
	"testing"
)

func TestVaultAuthenticationAndNonce(t *testing.T) {
	v, err := NewVault(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatal(err)
	}
	ct, n := v.Seal("secret-marker", "installation:secret:1")
	plain, err := v.Open(ct, n, "installation:secret:1")
	if err != nil || string(plain) != "secret-marker" {
		t.Fatal("roundtrip failed")
	}
	if bytes.Contains(ct, []byte("secret-marker")) {
		t.Fatal("plaintext exposed")
	}
	if _, err = v.Open(ct, n, "other:secret:1"); err == nil {
		t.Fatal("AAD binding missing")
	}
	ct[0] ^= 1
	if _, err = v.Open(ct, n, "installation:secret:1"); err == nil {
		t.Fatal("tamper accepted")
	}
	_, other := v.Seal("secret-marker", "installation:secret:1")
	if bytes.Equal(n, other) {
		t.Fatal("nonce reused")
	}
}
func TestIdentifiers(t *testing.T) {
	if !UUID(ID()) || UUID("../../etc/passwd") {
		t.Fatal("invalid UUID behavior")
	}
	a, b := Token(), Token()
	if len(a) < 43 || a == b {
		t.Fatal("weak proof")
	}
	if Equal(Hash(a), Hash(b)) {
		t.Fatal("proof collision")
	}
}
