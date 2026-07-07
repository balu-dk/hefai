package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashRoundtrip(t *testing.T) {
	hash, err := HashPassword("hemmeligt-kodeord")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "hemmeligt-kodeord") {
		t.Error("correct password rejected")
	}
	if CheckPassword(hash, "forkert") {
		t.Error("wrong password accepted")
	}
}

func TestTokenRoundtrip(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", time.Hour)
	id := uuid.New()

	token, err := issuer.Issue(id)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := issuer.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != id {
		t.Errorf("parsed %v, want %v", parsed, id)
	}
}

func TestTokenRejectsWrongSecret(t *testing.T) {
	token, err := NewTokenIssuer("secret-a", time.Hour).Issue(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewTokenIssuer("secret-b", time.Hour).Parse(token); err == nil {
		t.Error("token signed with different secret accepted")
	}
}

func TestTokenRejectsExpired(t *testing.T) {
	issuer := NewTokenIssuer("secret", -time.Minute)
	token, err := issuer.Issue(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Parse(token); err == nil {
		t.Error("expired token accepted")
	}
}
