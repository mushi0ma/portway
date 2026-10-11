package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
)

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func UUID(s string) bool { return uuidRE.MatchString(s) }
func ID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("entropy unavailable")
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func Token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("entropy unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func Hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func MAC(key []byte, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
func Equal(a, b string) bool { return hmac.Equal([]byte(a), []byte(b)) }

type Vault struct{ aead cipher.AEAD }

func NewVault(key []byte) (*Vault, error) {
	if len(key) != 32 {
		return nil, errors.New("AES-256-GCM requires a 32-byte key")
	}
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	a, err := cipher.NewGCM(c)
	return &Vault{a}, err
}
func (v *Vault) Seal(value, aad string) (ciphertext, nonce []byte) {
	nonce = make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic("entropy unavailable")
	}
	return v.aead.Seal(nil, nonce, []byte(value), []byte(aad)), nonce
}
func (v *Vault) Open(ciphertext, nonce []byte, aad string) ([]byte, error) {
	return v.aead.Open(nil, nonce, ciphertext, []byte(aad))
}
