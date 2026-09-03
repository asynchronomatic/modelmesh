package modeldex

// OpenaiModel represents metadata for an OpenAI model, including ID, creation time, owner, and context-related details.
type OpenaiModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
	// Extensions from other providers[ Grok, ]
	ContextLength        int `json:"context_length"`
	LongContextThreshold int `json:"long_context_threshold"`
}

// OpenaiModelList represents a collection of OpenAI models and the associated parent object metadata.
type OpenaiModelList struct {
	Object string        `json:"object"`
	Data   []OpenaiModel `json:"data"`
}
