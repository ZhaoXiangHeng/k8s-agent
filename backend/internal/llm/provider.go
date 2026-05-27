package llm

type Protocol string

const (
	ProtocolOpenAI    Protocol = "openai"
	ProtocolAnthropic Protocol = "anthropic"
	ProtocolMock      Protocol = "mock"
)

type ProviderConfig struct {
	ID       string
	Name     string
	Protocol Protocol
	BaseURL  string
	APIKey   string
	Enabled  bool
}

type ModelBinding struct {
	ModelID           string
	DisplayName       string
	ProviderID        string
	IsDefault         bool
	SupportsTools     bool
	SupportsStreaming bool
}

type ChatRequest struct {
	ModelID string
	System  string
	Message string
	Tools   []ToolDefinition
}

type ToolDefinition struct {
	Name        string
	Description string
}

type ChatResponse struct {
	Content   string
	ToolCall  *ToolCall
	RawResult string
}

type ToolCall struct {
	Name      string
	Arguments map[string]string
}

func SelectDefaultModel(models []ModelBinding) (ModelBinding, bool) {
	if len(models) == 0 {
		return ModelBinding{}, false
	}
	for _, model := range models {
		if model.IsDefault {
			return model, true
		}
	}
	return models[0], true
}
