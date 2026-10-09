package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	defaultMemoryKiB   = 19 * 1024
	defaultIterations  = 2
	defaultParallelism = 1
	defaultSaltLength  = 16
	defaultKeyLength   = 32

	minMemoryKiB   = 64
	maxMemoryKiB   = 64 * 1024
	minIterations  = 1
	maxIterations  = 10
	minParallelism = 1
	maxParallelism = 16
	minSaltLength  = 8
	maxSaltLength  = 64
	minKeyLength   = 16
	maxKeyLength   = 64
	minConcurrent  = 1
	maxConcurrent  = 64

	dummyPassword = "badgoodtune-dummy-password"
)

type Argon2Params struct {
	MemoryKiB     uint32
	Iterations    uint32
	Parallelism   uint8
	SaltLength    uint32
	KeyLength     uint32
	MaxConcurrent int
}

func DefaultArgon2Params(maxConcurrent int) Argon2Params {
	return Argon2Params{
		MemoryKiB:     defaultMemoryKiB,
		Iterations:    defaultIterations,
		Parallelism:   defaultParallelism,
		SaltLength:    defaultSaltLength,
		KeyLength:     defaultKeyLength,
		MaxConcurrent: maxConcurrent,
	}
}

type Argon2Hasher struct {
	params    Argon2Params
	semaphore chan struct{}
	dummyHash string
}

func NewArgon2Hasher(params Argon2Params) (*Argon2Hasher, error) {
	if err := validateArgon2Params(params); err != nil {
		return nil, err
	}
	hasher := &Argon2Hasher{params: params, semaphore: make(chan struct{}, params.MaxConcurrent)}
	dummySalt := make([]byte, params.SaltLength)
	dummyKey := argon2.IDKey([]byte(dummyPassword), dummySalt, params.Iterations, params.MemoryKiB, params.Parallelism, params.KeyLength)
	hasher.dummyHash = encodeArgon2Hash(params, dummySalt, dummyKey)
	return hasher, nil
}

func (h *Argon2Hasher) Hash(ctx context.Context, password string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()

	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, h.params.Iterations, h.params.MemoryKiB, h.params.Parallelism, h.params.KeyLength)
	return encodeArgon2Hash(h.params, salt, key), nil
}

func (h *Argon2Hasher) Matches(ctx context.Context, encodedHash, password string) (bool, error) {
	params, salt, expected, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return false, err
	}
	if err := h.acquire(ctx); err != nil {
		return false, err
	}
	defer h.release()

	actual := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKiB, params.Parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func (h *Argon2Hasher) DummyHash() string {
	return h.dummyHash
}

func (h *Argon2Hasher) acquire(ctx context.Context) error {
	select {
	case h.semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Argon2Hasher) release() {
	<-h.semaphore
}

func validateArgon2Params(params Argon2Params) error {
	if params.MemoryKiB < minMemoryKiB || params.MemoryKiB > maxMemoryKiB {
		return fmt.Errorf("Argon2 memory must be between %d KiB and %d KiB", minMemoryKiB, maxMemoryKiB)
	}
	if params.Iterations < minIterations || params.Iterations > maxIterations {
		return fmt.Errorf("Argon2 iterations must be between %d and %d", minIterations, maxIterations)
	}
	if params.Parallelism < minParallelism || params.Parallelism > maxParallelism {
		return fmt.Errorf("Argon2 parallelism must be between %d and %d", minParallelism, maxParallelism)
	}
	if params.SaltLength < minSaltLength || params.SaltLength > maxSaltLength {
		return fmt.Errorf("Argon2 salt length must be between %d and %d bytes", minSaltLength, maxSaltLength)
	}
	if params.KeyLength < minKeyLength || params.KeyLength > maxKeyLength {
		return fmt.Errorf("Argon2 key length must be between %d and %d bytes", minKeyLength, maxKeyLength)
	}
	if params.MaxConcurrent < minConcurrent || params.MaxConcurrent > maxConcurrent {
		return fmt.Errorf("Argon2 concurrency must be between %d and %d", minConcurrent, maxConcurrent)
	}
	return nil
}

func encodeArgon2Hash(params Argon2Params, salt, key []byte) string {
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.MemoryKiB,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

func decodeArgon2Hash(encoded string) (Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, errors.New("invalid Argon2id hash format")
	}
	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return Argon2Params{}, nil, nil, errors.New("unsupported Argon2 version")
	}
	params := Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.MemoryKiB, &params.Iterations, &params.Parallelism); err != nil ||
		parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", params.MemoryKiB, params.Iterations, params.Parallelism) {
		return Argon2Params{}, nil, nil, errors.New("invalid Argon2 parameters")
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("decode Argon2 salt: %w", err)
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("decode Argon2 key: %w", err)
	}
	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(key))
	params.MaxConcurrent = 1
	if err := validateArgon2Params(params); err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("unsafe Argon2 hash parameters: %w", err)
	}
	return params, salt, key, nil
}
