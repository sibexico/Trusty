package storage

import (
	"encoding/json"
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
	appDir := filepath.Join(configDir, "SecureMessenger")
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

	if err := json.NewDecoder(file).Decode(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Save() error {
	file, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(s)
}

// Adds a new contact and saves the store.
func (s *Store) AddContact(contact *Contact) {
	if s.Contacts[contact.Name] == nil {
		s.Contacts[contact.Name] = contact
		s.Messages[contact.Name] = []*Message{}
		s.Save()
	}
}

func (s *Store) AddMessage(contactName string, message *Message) {
	s.Messages[contactName] = append(s.Messages[contactName], message)
	s.Save()
}
