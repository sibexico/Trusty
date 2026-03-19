package storage

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

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
}

// Creates and loads a new store from the config file path.
func NewStore() (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(configDir, "Trusty")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return nil, err
	}
	filePath := filepath.Join(appDir, "data.json")

	s := &Store{
		Contacts: make(map[string]*Contact),
		Messages: make(map[string][]*Message),
		path:     filePath,
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(s); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if s.Contacts == nil {
		s.Contacts = make(map[string]*Contact)
	}
	if s.Messages == nil {
		s.Messages = make(map[string][]*Message)
	}
	return s, nil
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

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s); err != nil {
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
