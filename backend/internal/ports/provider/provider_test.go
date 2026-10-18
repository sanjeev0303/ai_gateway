package provider

import (
	"errors"
	"testing"
)

func TestModelCapabilities_HasCapability(t *testing.T) {
	caps := ModelCapabilities{
		ModelID: "gpt-4o",
		Capabilities: []Capability{
			CapabilityTextGeneration,
			CapabilityStreaming,
			CapabilityToolCalling,
			CapabilityMultimodal,
		},
	}

	if !caps.HasCapability(CapabilityTextGeneration) {
		t.Errorf("expected text_generation to be present")
	}
	if !caps.HasCapability(CapabilityStreaming) {
		t.Errorf("expected streaming to be present")
	}
	if caps.HasCapability(CapabilityEmbeddings) {
		t.Errorf("expected embeddings to be absent")
	}
}

func TestModelCapabilities_ValidateRequest(t *testing.T) {
	caps := ModelCapabilities{
		ModelID:           "test-model",
		SupportsStreaming: true,
		SupportsTools:     false,
		SupportsJSON:      false,
		SupportsVision:    false,
		Capabilities: []Capability{
			CapabilityTextGeneration,
			CapabilityStreaming,
		},
	}

	// 1. Valid request
	validReq := NormalizedRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		Stream: true,
	}
	if err := caps.ValidateRequest(validReq); err != nil {
		t.Fatalf("expected valid request to pass, got: %v", err)
	}

	// 2. Unsupported tool calling
	reqWithTools := validReq
	reqWithTools.Tools = []ToolDefinition{{Name: "calculator"}}
	err := caps.ValidateRequest(reqWithTools)
	if err == nil || !errors.Is(err, ErrUnsupportedCapability) {
		t.Errorf("expected ErrUnsupportedCapability for tools, got: %v", err)
	}

	// 3. Unsupported structured JSON
	reqWithJSON := validReq
	reqWithJSON.ResponseFormat = &ResponseFormat{Type: "json_object"}
	err = caps.ValidateRequest(reqWithJSON)
	if err == nil || !errors.Is(err, ErrUnsupportedCapability) {
		t.Errorf("expected ErrUnsupportedCapability for JSON, got: %v", err)
	}

	// 4. Unsupported multimodal images
	reqWithImage := validReq
	reqWithImage.Messages = []Message{
		{
			Role: RoleUser,
			Parts: []ContentPart{
				{Type: ContentTypeImage, ImageURL: "http://example.com/pic.png"},
			},
		},
	}
	err = caps.ValidateRequest(reqWithImage)
	if err == nil || !errors.Is(err, ErrUnsupportedCapability) {
		t.Errorf("expected ErrUnsupportedCapability for vision, got: %v", err)
	}

	// 5. Model that doesn't support streaming
	capsNoStream := caps
	capsNoStream.SupportsStreaming = false
	capsNoStream.Capabilities = []Capability{CapabilityTextGeneration}
	err = capsNoStream.ValidateRequest(validReq)
	if err == nil || !errors.Is(err, ErrUnsupportedCapability) {
		t.Errorf("expected ErrUnsupportedCapability for streaming, got: %v", err)
	}
}
