package security

import (
	"crypto/rand"
	"crypto/sha256"
)

type SessionTokenSource struct{}

func (SessionTokenSource) New() (string, []byte, error) {
	token := rand.Text()
	return token, digestToken(token), nil
}

func (SessionTokenSource) Digest(token string) []byte {
	return digestToken(token)
}

func digestToken(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}
