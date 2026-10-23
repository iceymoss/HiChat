package models

import (
	"encoding/json"
	"testing"
)

func TestAuthenticatedMessageUsesServerIdentity(t *testing.T) {
	data, err := authenticatedMessage([]byte(`{"userId":999,"targetId":2,"Type":1,"Content":"hello"}`), 7)
	if err != nil {
		t.Fatal(err)
	}

	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatal(err)
	}
	if message.FormId != 7 {
		t.Fatalf("expected authenticated user 7, got %d", message.FormId)
	}
	if message.MessageID == "" {
		t.Fatal("expected a server-generated message ID")
	}
}

func TestAuthenticatedMessageRejectsInvalidRouting(t *testing.T) {
	testCases := []string{
		`{"targetId":0,"Type":1}`,
		`{"targetId":2,"Type":99}`,
	}
	for _, testCase := range testCases {
		if _, err := authenticatedMessage([]byte(testCase), 7); err == nil {
			t.Fatalf("expected invalid message to be rejected: %s", testCase)
		}
	}
}

func TestAuthenticatedMessageAllowsHeartbeat(t *testing.T) {
	data, err := authenticatedMessage([]byte(`{"Type":0,"Content":"heartbeat"}`), 7)
	if err != nil {
		t.Fatal(err)
	}

	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatal(err)
	}
	if message.FormId != 7 {
		t.Fatalf("expected authenticated user 7, got %d", message.FormId)
	}
}
