package llm

import "testing"

func TestSelectDefaultModelReturnsExplicitDefault(t *testing.T) {
	models := []ModelBinding{
		{ModelID: "claude", DisplayName: "Claude", IsDefault: false},
		{ModelID: "gpt", DisplayName: "GPT", IsDefault: true},
	}

	model, ok := SelectDefaultModel(models)

	if !ok {
		t.Fatal("expected a default model")
	}
	if model.ModelID != "gpt" {
		t.Fatalf("expected gpt, got %s", model.ModelID)
	}
}

func TestSelectDefaultModelFallsBackToFirstModel(t *testing.T) {
	models := []ModelBinding{
		{ModelID: "claude", DisplayName: "Claude"},
		{ModelID: "gpt", DisplayName: "GPT"},
	}

	model, ok := SelectDefaultModel(models)

	if !ok {
		t.Fatal("expected first model fallback")
	}
	if model.ModelID != "claude" {
		t.Fatalf("expected claude, got %s", model.ModelID)
	}
}
