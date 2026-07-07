// Package ai defines the provider-agnostic LLM interface. The assistant is
// built against Provider so the concrete backend (Claude API, local model,
// …) can be chosen and swapped without touching the RAG or service layers.
//
// The provider is deliberately NOT a calculation engine: it structures,
// explains and summarises grounded source material. All numbers presented
// as results must come from the deterministic calc package.
package ai

import (
	"context"
	"errors"
)

// ErrNotConfigured signals that no LLM provider is set up. Callers degrade
// gracefully: retrieval still works, only the generated prose is missing.
var ErrNotConfigured = errors.New("ingen LLM-provider er konfigureret")

type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

type Request struct {
	System   string
	Messages []Message
	// MaxTokens bounds the response length; 0 means provider default.
	MaxTokens int
}

type Provider interface {
	// Complete returns the assistant's next message for the conversation.
	Complete(ctx context.Context, req Request) (string, error)
	// Name identifies the provider in logs and API responses.
	Name() string
}

// Unconfigured is the default provider until one is chosen. Every call
// fails with ErrNotConfigured.
type Unconfigured struct{}

func (Unconfigured) Complete(context.Context, Request) (string, error) {
	return "", ErrNotConfigured
}

func (Unconfigured) Name() string { return "none" }
