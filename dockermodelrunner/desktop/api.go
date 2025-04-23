package desktop

// ProgressMessage represents a message sent during model pull operations
type ProgressMessage struct {
	Type    string `json:"type"`    // "progress", "success", or "error"
	Message string `json:"message"` // Human-readable message
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []OpenAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
			Role    string `json:"role,omitempty"`
		} `json:"delta"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type OpenAIModelList struct {
	Object string         `json:"object"`
	Data   []*OpenAIModel `json:"data"`
}

// TODO: To be replaced by the Model struct from pianta's common/pkg/inference/models/api.go.
// (https://github.com/docker/pinata/pull/33331)
type Format string

type Config struct {
	Format       Format `json:"format,omitempty"`
	Quantization string `json:"quantization,omitempty"`
	Parameters   string `json:"parameters,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Size         string `json:"size,omitempty"`
}

type Model struct {
	// ID is the globally unique model identifier.
	ID string `json:"id"`
	// Tags are the list of tags associated with the model.
	Tags []string `json:"tags"`
	// Created is the Unix epoch timestamp corresponding to the model creation.
	Created int64 `json:"created"`
	// Config describes the model.
	Config Config `json:"config"`
}
