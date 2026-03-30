package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"math/big"
)

var (
	dhPrime     = mustParseHexInt("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA237327FFFFFFFFFFFFFFFF")
	dhGenerator = big.NewInt(2)
	one         = big.NewInt(1)
)

func mustParseHexInt(value string) *big.Int {
	n, ok := new(big.Int).SetString(value, 16)
	if !ok {
		panic("invalid DH prime")
	}
	return n
}

func GenerateDHKeyPair() (privateKey *big.Int, publicKey string, err error) {
	max := new(big.Int).Sub(dhPrime, big.NewInt(3))
	privateKey, err = rand.Int(rand.Reader, max)
	if err != nil {
		return nil, "", err
	}
	privateKey.Add(privateKey, big.NewInt(2))

	publicKeyBytes := new(big.Int).Exp(dhGenerator, privateKey, dhPrime).Bytes()
	return privateKey, base64.StdEncoding.EncodeToString(publicKeyBytes), nil
}

func ComputeSharedSecret(privateKey *big.Int, otherPublicKeyB64 string) ([]byte, error) {
	if privateKey == nil || privateKey.Sign() <= 0 {
		return nil, errors.New("invalid private key")
	}

	otherPublicKeyBytes, err := base64.StdEncoding.DecodeString(otherPublicKeyB64)
	if err != nil {
		return nil, errors.New("invalid base64 for public key")
	}
	if len(otherPublicKeyBytes) == 0 {
		return nil, errors.New("public key cannot be empty")
	}

	otherPublicKeyInt := new(big.Int).SetBytes(otherPublicKeyBytes)
	if otherPublicKeyInt.Cmp(one) <= 0 || otherPublicKeyInt.Cmp(new(big.Int).Sub(dhPrime, one)) >= 0 {
		return nil, errors.New("public key is out of valid range")
	}

	sharedSecret := new(big.Int).Exp(otherPublicKeyInt, privateKey, dhPrime)
	if sharedSecret.Sign() == 0 {
		return nil, errors.New("shared secret generation failed")
	}

	primeLen := (dhPrime.BitLen() + 7) / 8
	sharedSecretBytes := sharedSecret.FillBytes(make([]byte, primeLen))
	key := sha256.Sum256(sharedSecretBytes)
	return key[:], nil
}

func Encrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertextB64 string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("decryption failed (invalid key or tampered message)")
	}
	return string(plaintext), nil
}

func GenerateAuthCode(sharedKey []byte, preSharedSecret string) string {
	mac := hmac.New(sha256.New, sharedKey)
	mac.Write([]byte(preSharedSecret))
	fullCode := mac.Sum(nil)

	hexCode := hex.EncodeToString(fullCode[:6])
	return hexCode[:4] + "-" + hexCode[4:8] + "-" + hexCode[8:12]
}
