package domain

import "strings"

// Model families recognised by name alone. The gateway asks the model registry
// first; these rules cover models the registry has not synced. Every caller
// that needs to know a model's provider goes through ProviderForModel, so a
// new family is added here once.
var (
	openAIModelPrefixes  = []string{"gpt-", "o1", "o3", "text-embedding", "dall-e"}
	mistralModelPrefixes = []string{"mistral", "open-mistral-", "codestral-", "ministral-", "magistral-", "pixtral-", "devstral-", "voxtral-"}
)

// ProviderForModel returns the provider that serves a model, judged from its
// name, or "" when the name does not say. Mistral's own models are reached
// through a custom OpenAI-compatible connection. Anything with a slash is a
// Hugging Face repo id, including "mistralai/...".
func ProviderForModel(model string) string {
	switch {
	case model == "":
		return ""
	case strings.HasPrefix(model, "claude-"):
		return ProviderKindAnthropic
	case hasAnyPrefix(model, openAIModelPrefixes):
		return ProviderKindOpenAI
	case strings.HasPrefix(model, "gemini-"):
		return ProviderKindGemini
	case strings.HasPrefix(model, "hf-") || strings.Contains(model, "/"):
		return ProviderKindHuggingFace
	case IsMistralModel(model):
		return ProviderKindCustom
	}
	return ""
}

// IsMistralModel reports whether a slug names one of Mistral's own models.
func IsMistralModel(slug string) bool {
	slug = strings.ToLower(slug)
	return !strings.Contains(slug, "/") && hasAnyPrefix(slug, mistralModelPrefixes)
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
