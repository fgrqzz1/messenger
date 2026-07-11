package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

// Hash returns an Argon2id hash in PHC string format.
func Hash(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64Salt, b64Hash), nil
}

// Verify checks a password against an Argon2id PHC hash.
func Verify(password, encoded string) (bool, error) {
	salt, hash, params, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}

	other := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(hash)))
	if subtle.ConstantTimeCompare(hash, other) == 1 {
		return true, nil
	}

	return false, nil
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

func decodeHash(encoded string) (salt, hash []byte, params argonParams, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return nil, nil, params, errors.New("password: invalid hash format")
	}
	if parts[1] != "argon2id" {
		return nil, nil, params, errors.New("password: unsupported algorithm")
	}

	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, params, fmt.Errorf("password: parse version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, params, errors.New("password: incompatible argon2 version")
	}

	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.time, &params.threads); err != nil {
		return nil, nil, params, fmt.Errorf("password: parse params: %w", err)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, params, fmt.Errorf("password: decode salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, params, fmt.Errorf("password: decode hash: %w", err)
	}

	return salt, hash, params, nil
}
