package backend

import "context"

// Fake est un backend déterministe pour les tests et le développement hors GPU.
type Fake struct {
	NameValue string
	Models    []string
	// Tokens émis comme fragments successifs lors de Generate.
	Tokens []string
	// FailWith, s'il est non nil, est renvoyé par Generate.
	FailWith error
}

// NewFake crée un backend factice qui répond avec les tokens donnés.
func NewFake(models []string, tokens ...string) *Fake {
	return &Fake{NameValue: "fake", Models: models, Tokens: tokens}
}

func (f *Fake) Name() string { return f.NameValue }

func (f *Fake) Available(context.Context) error { return nil }

func (f *Fake) ListModels(context.Context) ([]string, error) {
	return append([]string(nil), f.Models...), nil
}

func (f *Fake) Generate(ctx context.Context, req GenerateRequest, onChunk func(Chunk) error) (Usage, error) {
	if f.FailWith != nil {
		return Usage{}, f.FailWith
	}
	for _, tok := range f.Tokens {
		select {
		case <-ctx.Done():
			return Usage{}, ctx.Err()
		default:
		}
		if err := onChunk(Chunk{Content: tok}); err != nil {
			return Usage{}, err
		}
	}
	if err := onChunk(Chunk{Done: true}); err != nil {
		return Usage{}, err
	}
	return Usage{PromptTokens: 1, CompletionTokens: len(f.Tokens)}, nil
}

var _ Backend = (*Fake)(nil)
