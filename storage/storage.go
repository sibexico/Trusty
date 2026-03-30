package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

const (
	currentStorageVersion = 2
	legacyStorageVersion  = 1
	kdfIterations         = 120000
	kdfSaltBytes          = 16
)

var ErrInvalidPassphrase = errors.New("invalid passphrase or profile data")

// Holds the data for a single user contact.
type Contact struct {
	Name      string `json:"name"`
	SharedKey []byte `json:"shared_key"`
}

// Holds the data for a single message in the history.
type Message struct {
	Timestamp int64  `json:"timestamp"`
	IsSent    bool   `json:"is_sent"` // true if sent by me
	Content   string `json:"content"` // The decrypted content
}

// The main container for all application data.
type Store struct {
	Contacts map[string]*Contact   `json:"contacts"`
	Messages map[string][]*Message `json:"messages"`
	path     string
	secret   string
}

type persistedStore struct {
	Contacts map[string]*Contact   `json:"contacts"`
	Messages map[string][]*Message `json:"messages"`
}

type fileEnvelope struct {
	Version    int    `json:"version"`
	Iterations int    `json:"iterations,omitempty"`
	Salt       string `json:"salt,omitempty"`
	Nonce      string `json:"nonce,omitempty"`
	Payload    string `json:"payload"`
}

func ProfilesDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	profilesDir := filepath.Join(configDir, "Trusty", "profiles")
	if err := os.MkdirAll(profilesDir, 0700); err != nil {
		return "", err
	}
	return profilesDir, nil
}

func NewStore(profilePath string, passphrase string) (*Store, error) {
	if profilePath == "" {
		return nil, errors.New("profile path cannot be empty")
	}
	if passphrase == "" {
		return nil, errors.New("passphrase cannot be empty")
	}

	if err := os.MkdirAll(filepath.Dir(profilePath), 0700); err != nil {
		return nil, err
	}

	s := &Store{
		Contacts: make(map[string]*Contact),
		Messages: make(map[string][]*Message),
		path:     profilePath,
		secret:   passphrase,
	}

	file, err := os.Open(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	defer file.Close()

	rawData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if len(rawData) == 0 {
		return s, nil
	}

	loadedFromLegacy := false
	if err := s.loadV2(rawData); err != nil {
		if err := s.loadV1(rawData); err != nil {
			if err := s.loadLegacy(rawData); err != nil {
				return nil, err
			}
		}
		loadedFromLegacy = true
	}

	if loadedFromLegacy {
		if err := s.Save(); err != nil {
			return nil, err
		}
	}

	if s.Contacts == nil {
		s.Contacts = make(map[string]*Contact)
	}
	if s.Messages == nil {
		s.Messages = make(map[string][]*Message)
	}
	return s, nil
}

func (s *Store) loadLegacy(rawData []byte) error {
	legacy := &persistedStore{}
	if err := json.Unmarshal(rawData, legacy); err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	s.Contacts = legacy.Contacts
	s.Messages = legacy.Messages
	return nil
}

func (s *Store) loadV1(rawData []byte) error {
	envelope := &fileEnvelope{}
	if err := json.Unmarshal(rawData, envelope); err != nil {
		return err
	}
	if envelope.Version != legacyStorageVersion {
		return errors.New("unsupported storage version")
	}
	if envelope.Payload == "" {
		return errors.New("missing storage payload")
	}

	protectedData, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return err
	}

	plaintext, err := unprotectData(protectedData)
	if err != nil {
		return err
	}

	persisted := &persistedStore{}
	if err := json.Unmarshal(plaintext, persisted); err != nil {
		return err
	}

	s.Contacts = persisted.Contacts
	s.Messages = persisted.Messages
	return nil
}

func (s *Store) loadV2(rawData []byte) error {
	envelope := &fileEnvelope{}
	if err := json.Unmarshal(rawData, envelope); err != nil {
		return err
	}
	if envelope.Version != currentStorageVersion {
		return errors.New("unsupported storage version")
	}
	if envelope.Payload == "" || envelope.Salt == "" || envelope.Nonce == "" {
		return errors.New("missing storage payload")
	}

	iterations := envelope.Iterations
	if iterations <= 0 {
		iterations = kdfIterations
	}

	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil {
		return err
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return err
	}

	key := deriveKey(s.secret, salt, iterations)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ErrInvalidPassphrase
	}

	persisted := &persistedStore{}
	if err := json.Unmarshal(plaintext, persisted); err != nil {
		return err
	}

	s.Contacts = persisted.Contacts
	s.Messages = persisted.Messages
	return nil
}

func (s *Store) buildEncryptedPayload() ([]byte, error) {
	persisted := &persistedStore{
		Contacts: s.Contacts,
		Messages: s.Messages,
	}

	plaintext, err := json.Marshal(persisted)
	if err != nil {
		return nil, err
	}

	salt := make([]byte, kdfSaltBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key := deriveKey(s.secret, salt, kdfIterations)

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

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	envelope := &fileEnvelope{
		Version:    currentStorageVersion,
		Iterations: kdfIterations,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Payload:    base64.StdEncoding.EncodeToString(ciphertext),
	}

	return json.MarshalIndent(envelope, "", "  ")
}

func deriveKey(passphrase string, salt []byte, iterations int) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, iterations, 32, sha256.New)
}

func (s *Store) Save() error {
	if s.path == "" {
		return errors.New("storage path is not configured")
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
		os.Remove(tmpPath)
	}()

	payload, err := s.buildEncryptedPayload()
	if err != nil {
		return err
	}

	if _, err := file.Write(payload); err != nil {
		return err
	}
	if _, err := file.Write([]byte("\n")); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		if removeErr := os.Remove(s.path); removeErr != nil && !os.IsNotExist(removeErr) {
			return err
		}
		return os.Rename(tmpPath, s.path)
	}

	return nil
}

// Adds a new contact and saves the store.
func (s *Store) AddContact(contact *Contact) error {
	if contact == nil {
		return errors.New("contact cannot be nil")
	}
	if contact.Name == "" {
		return errors.New("contact name cannot be empty")
	}

	if s.Contacts[contact.Name] == nil {
		s.Contacts[contact.Name] = contact
		s.Messages[contact.Name] = []*Message{}
		if err := s.Save(); err != nil {
			delete(s.Contacts, contact.Name)
			delete(s.Messages, contact.Name)
			return err
		}
	}

	return nil
}

func (s *Store) AddMessage(contactName string, message *Message) error {
	if contactName == "" {
		return errors.New("contact name cannot be empty")
	}
	if message == nil {
		return errors.New("message cannot be nil")
	}
	if _, ok := s.Contacts[contactName]; !ok {
		return errors.New("contact does not exist")
	}

	s.Messages[contactName] = append(s.Messages[contactName], message)
	if err := s.Save(); err != nil {
		s.Messages[contactName] = s.Messages[contactName][:len(s.Messages[contactName])-1]
		return err
	}

	return nil
}
