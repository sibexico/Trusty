package storage

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestProtectDataRoundTrip(t *testing.T) {
	plain := []byte("hello trusty")

	protected, err := protectData(plain)
	if err != nil {
		t.Fatalf("protectData failed: %v", err)
	}

	decrypted, err := unprotectData(protected)
	if err != nil {
		t.Fatalf("unprotectData failed: %v", err)
	}

	if string(decrypted) != string(plain) {
		t.Fatalf("round-trip mismatch: got %q, want %q", string(decrypted), string(plain))
	}
}

func TestLoadV1PlaintextFallback(t *testing.T) {
	p := &persistedStore{
		Contacts: map[string]*Contact{
			"alice": {Name: "alice", SharedKey: []byte("k")},
		},
		Messages: map[string][]*Message{
			"alice": {{Timestamp: 1, IsSent: true, Content: "hi"}},
		},
	}

	plaintext, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal persistedStore failed: %v", err)
	}

	envelope := &fileEnvelope{
		Version: legacyStorageVersion,
		Payload: base64.StdEncoding.EncodeToString(plaintext),
	}

	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope failed: %v", err)
	}

	s := &Store{}
	if err := s.loadV1(raw); err != nil {
		t.Fatalf("loadV1 should accept legacy plaintext payload: %v", err)
	}

	if s.Contacts["alice"] == nil || s.Contacts["alice"].Name != "alice" {
		t.Fatalf("contact was not loaded correctly")
	}
	if len(s.Messages["alice"]) != 1 || s.Messages["alice"][0].Content != "hi" {
		t.Fatalf("messages were not loaded correctly")
	}
}
