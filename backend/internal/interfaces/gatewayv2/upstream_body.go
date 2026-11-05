package gatewayv2

import (
	"bytes"
	"encoding/json"
)

// gatewayOnlyFields are request fields the gateway reads for itself. Providers
// reject fields they do not know (Gemini answers 400), so they are removed
// before the request is forwarded.
var gatewayOnlyFields = []string{"conversation_id", "project_id"}

// redactSkipKeys hold identifiers rather than content and are never redacted.
var redactSkipKeys = map[string]bool{"model": true, "role": true, "type": true, "id": true, "tool_call_id": true}

// prepareUpstreamBody removes gateway-only fields and, when redact is set,
// masks every content string in the request. It returns nil when the body
// needs no change, so most requests are still forwarded byte for byte.
func prepareUpstreamBody(body []byte, redact func(string) string) []byte {
	hasGatewayFields := false
	for _, field := range gatewayOnlyFields {
		if bytes.Contains(body, []byte(`"`+field+`"`)) {
			hasGatewayFields = true
			break
		}
	}
	if !hasGatewayFields && redact == nil {
		return nil
	}

	// UseNumber keeps large integers such as a seed exact through the round trip.
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var req map[string]any
	if err := dec.Decode(&req); err != nil {
		return nil
	}

	for _, field := range gatewayOnlyFields {
		delete(req, field)
	}
	if redact != nil {
		for key, value := range req {
			if !redactSkipKeys[key] {
				req[key] = redactValue(value, redact)
			}
		}
	}

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(req); err != nil {
		return nil
	}
	return bytes.TrimRight(out.Bytes(), "\n")
}

func redactValue(value any, redact func(string) string) any {
	switch v := value.(type) {
	case string:
		return redact(v)
	case []any:
		for i := range v {
			v[i] = redactValue(v[i], redact)
		}
	case map[string]any:
		for key, inner := range v {
			if !redactSkipKeys[key] {
				v[key] = redactValue(inner, redact)
			}
		}
	}
	return value
}
