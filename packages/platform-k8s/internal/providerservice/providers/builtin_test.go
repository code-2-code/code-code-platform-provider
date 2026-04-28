package providers

import "testing"

func TestRegisterBuiltins(t *testing.T) {
	t.Parallel()

	lookup := NewLookup()
	if err := RegisterBuiltins(lookup); err != nil {
		t.Fatalf("RegisterBuiltins() error = %v", err)
	}

	provider, err := lookup.Get("openai-compatible")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if provider == nil {
		t.Fatal("Get() returned nil provider")
	}

	provider, err = lookup.Get("gemini")
	if err != nil {
		t.Fatalf("Get() gemini error = %v", err)
	}
	if provider == nil {
		t.Fatal("Get() gemini returned nil provider")
	}

	provider, err = lookup.Get("anthropic")
	if err != nil {
		t.Fatalf("Get() anthropic error = %v", err)
	}
	if provider == nil {
		t.Fatal("Get() anthropic returned nil provider")
	}
}
