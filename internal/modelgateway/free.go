package modelgateway

import (
	"encoding/json"
	"portway/internal/openrouter"
	"portway/internal/problem"
)

// Fixed routing policy is part of the counted request too. There is no fallback
// model, plugin, web search, media generation, BYOK, or optional paid feature.
func freeBody(body []byte) ([]byte, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil, failure("MODEL_REQUEST_INVALID")
	}
	allowed := map[string]bool{"model": true, "messages": true, "tools": true, "tool_choice": true, "parallel_tool_calls": true, "stream": true, "stream_options": true, "max_tokens": true, "n": true, "provider": true}
	for key := range fields {
		if !allowed[key] {
			return nil, problem.New(403, "FREE_MODEL_REQUIRED", "Paid or unknown model request options are disabled")
		}
	}
	var model string
	if json.Unmarshal(fields["model"], &model) != nil || !openrouter.FreeID(model) {
		return nil, problem.New(403, "FREE_MODEL_REQUIRED", "An explicit free model is required")
	}
	provider, _ := json.Marshal(map[string]any{"allow_fallbacks": false, "require_parameters": true, "max_price": map[string]string{"prompt": "0", "completion": "0", "request": "0", "image": "0", "audio": "0"}})
	fields["provider"] = provider
	// OpenRouter reports usage automatically. These optional OpenAI fields are
	// absent from the selected free providers' supported parameter sets.
	// The protocol parser still rejects multiple choices/tool calls.
	delete(fields, "stream_options")
	delete(fields, "parallel_tool_calls")
	delete(fields, "n")
	if string(fields["tool_choice"]) == `"none"` {
		delete(fields, "tool_choice")
		delete(fields, "tools")
	}
	return json.Marshal(fields)
}
