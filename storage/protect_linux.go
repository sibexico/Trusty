//go:build linux

package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const linuxProtectedDataVersion byte = 1

var linuxMachineIDCandidates = []string{
	"/etc/machine-id",
	"/var/lib/dbus/machine-id",
}

func linuxProtectionKey() ([]byte, error) {
	machineID, err := readLinuxMachineID()
	if err != nil {
		return nil, err
	}

	uid := strconv.Itoa(os.Getuid())
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	digest := sha256.Sum256([]byte("trusty-linux-protect-v1|" + machineID + "|" + uid + "|" + homeDir))
	return digest[:], nil
}

func readLinuxMachineID() (string, error) {
	for _, candidate := range linuxMachineIDCandidates {
		raw, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		machineID := strings.TrimSpace(string(raw))
		if machineID != "" {
			return machineID, nil
		}
	}

	return "", errors.New("linux machine-id is unavailable")
}

func protectData(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return []byte{}, nil
	}

	key, err := linuxProtectionKey()
	if err != nil {
		return nil, fmt.Errorf("linux protect key failed: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plain, nil)

	out := make([]byte, 1+len(nonce)+len(ciphertext))
	out[0] = linuxProtectedDataVersion
	copy(out[1:], nonce)
	copy(out[1+len(nonce):], ciphertext)
	return out, nil
}

func unprotectData(protected []byte) ([]byte, error) {
	if len(protected) == 0 {
		return []byte{}, nil
	}

	if protected[0] != linuxProtectedDataVersion {
		return nil, errors.New("unsupported linux protected data version")
	}

	key, err := linuxProtectionKey()
	if err != nil {
		return nil, fmt.Errorf("linux unprotect key failed: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(protected) < 1+nonceSize {
		return nil, errors.New("invalid linux protected data")
	}

	nonce := protected[1 : 1+nonceSize]
	ciphertext := protected[1+nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("linux data unprotect failed")
	}

	copyOut := make([]byte, len(plaintext))
	copy(copyOut, plaintext)
	return copyOut, nil
}
