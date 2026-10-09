package usecase

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

const (
	DefaultPageSize           = 20
	MaxPageSize               = 100
	MaxPageTokenLen           = 2048
	pageTokenVersion          = 1
	pageTokenFingerprintBytes = 16
	canonicalUUIDLength       = 36
	firstUUIDHyphen           = 8
	secondUUIDHyphen          = 13
	thirdUUIDHyphen           = 18
	fourthUUIDHyphen          = 23
)

type pageToken struct {
	Version     int        `json:"v"`
	Resource    string     `json:"r"`
	Fingerprint string     `json:"f"`
	ID          string     `json:"id"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	ReleaseYear *int       `json:"releaseYear,omitempty"`
	Rating      *float64   `json:"rating,omitempty"`
	Score       *int       `json:"score,omitempty"`
}

func normalizePageSize(pageSize int) (int, error) {
	if pageSize < 0 {
		return 0, domain.NewValidationError(domain.Violation{Field: "pageSize", Description: "must not be negative"})
	}
	if pageSize == 0 {
		return DefaultPageSize, nil
	}
	if pageSize > MaxPageSize {
		return MaxPageSize, nil
	}
	return pageSize, nil
}

func fingerprint(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:pageTokenFingerprintBytes])
}

func encodePageToken(token pageToken) (string, error) {
	encoded, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodePageToken(raw string) (pageToken, error) {
	if len(raw) == 0 || len(raw) > MaxPageTokenLen {
		return pageToken{}, errors.New("page token has invalid length")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return pageToken{}, fmt.Errorf("decode base64: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(decoded))
	decoder.DisallowUnknownFields()
	var token pageToken
	if err := decoder.Decode(&token); err != nil {
		return pageToken{}, fmt.Errorf("decode JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return pageToken{}, err
	}
	if token.Version != pageTokenVersion || !validPageTokenID(token.ID) {
		return pageToken{}, errors.New("unsupported or incomplete page token")
	}
	return token, nil
}

func validPageTokenID(value string) bool {
	if len(value) != canonicalUUIDLength ||
		value[firstUUIDHyphen] != '-' ||
		value[secondUUIDHyphen] != '-' ||
		value[thirdUUIDHyphen] != '-' ||
		value[fourthUUIDHyphen] != '-' {
		return false
	}
	compact := value[:firstUUIDHyphen] +
		value[firstUUIDHyphen+1:secondUUIDHyphen] +
		value[secondUUIDHyphen+1:thirdUUIDHyphen] +
		value[thirdUUIDHyphen+1:fourthUUIDHyphen] +
		value[fourthUUIDHyphen+1:]
	_, err := hex.DecodeString(compact)
	return err == nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("page token contains multiple JSON values")
		}
		return fmt.Errorf("read page token suffix: %w", err)
	}
	return nil
}

func invalidPageToken() error {
	return domain.NewValidationError(domain.Violation{Field: "pageToken", Description: "is invalid or does not match the request"})
}
