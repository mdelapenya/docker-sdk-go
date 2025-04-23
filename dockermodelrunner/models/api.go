package models

import "github.com/docker/docker-sdk-go/dockermodelrunner/modeldistribution/types"

// ModelCreateRequest represents a model create request. It is designed to
// follow Docker Engine API conventions, most closely following the request
// associated with POST /images/create. At the moment is only designed to
// facilitate pulls, though in the future it may facilitate model building and
// refinement (such as fine tuning, quantization, or distillation).
type ModelCreateRequest struct {
	// From is the name of the model to pull.
	From string `json:"from"`
}

// ToOpenAI converts a types.Model to its OpenAI API representation.
func ToOpenAI(m *types.Model) *OpenAIModel {
	return &OpenAIModel{
		ID:      m.Tags[0],
		Object:  "model",
		Created: m.Created,
		OwnedBy: "docker",
	}
}

// ModelList represents a list of models.
type ModelList []*types.Model

// ToOpenAI converts the model list to its OpenAI API representation. This function never
// returns a nil slice (though it may return an empty slice).
func (l ModelList) ToOpenAI() *OpenAIModelList {
	// Convert the constituent models.
	models := make([]*OpenAIModel, len(l))
	for m, model := range l {
		models[m] = ToOpenAI(model)
	}

	// Create the OpenAI model list.
	return &OpenAIModelList{
		Object: "list",
		Data:   models,
	}
}

// OpenAIModel represents a locally stored model using OpenAI conventions.
type OpenAIModel struct {
	// ID is the model tag.
	ID string `json:"id"`
	// Object is the object type. For OpenAIModel, it is always "model".
	Object string `json:"object"`
	// Created is the Unix epoch timestamp corresponding to the model creation.
	Created int64 `json:"created"`
	// OwnedBy is the model owner. At the moment, it is always "docker".
	OwnedBy string `json:"owned_by"`
}

// OpenAIModelList represents a list of models using OpenAI conventions.
type OpenAIModelList struct {
	// Object is the object type. For OpenAIModelList, it is always "list".
	Object string `json:"object"`
	// Data is the list of models.
	Data []*OpenAIModel `json:"data"`
}
