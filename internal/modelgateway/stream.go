package modelgateway

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"

	"portway/internal/agents"
	"portway/internal/execution"
	"portway/internal/httpjson"
	"portway/internal/openrouter"
)

type usage struct {
	Input  *int            `json:"prompt_tokens"`
	Output *int            `json:"completion_tokens"`
	Total  *int            `json:"total_tokens"`
	Cost   json.RawMessage `json:"cost"`
}
type deltaTool struct {
	Index    *int   `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
type chunk struct {
	ID      string          `json:"id"`
	Error   json.RawMessage `json:"error"`
	Choices []struct {
		Index  *int    `json:"index"`
		Finish *string `json:"finish_reason"`
		Delta  struct {
			Role    string          `json:"role"`
			Content string          `json:"content"`
			Refusal string          `json:"refusal"`
			Legacy  json.RawMessage `json:"function_call"`
			Tools   []deltaTool     `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *usage `json:"usage"`
}

func readStream(reader io.Reader, r execution.Run, step execution.Step) (execution.ModelReply, error) {
	return readStreamPolicy(reader, r, step, false)
}
func readStreamPolicy(reader io.Reader, r execution.Run, step execution.Step, freeOnly bool) (execution.ModelReply, error) {
	limited := &io.LimitedReader{R: reader, N: (1 << 20) + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), 131072)
	var text, args, name, event strings.Builder
	id, callID, finish := "", "", ""
	toolSeen, done, usageSeen := false, false, false
	reply := execution.ModelReply{UsageSource: "unknown"}
	recordUsage := func(u *usage) error {
		if usageSeen || finish == "" || u.Input == nil || u.Output == nil || u.Total == nil || *u.Input < 0 || *u.Input > 100000000 || *u.Output < 0 || *u.Output > 100000000 || *u.Total != *u.Input+*u.Output {
			return failure("MODEL_USAGE_INVALID")
		}
		if freeOnly && !openrouter.Zero(u.Cost) {
			return failure("MODEL_COST_UNVERIFIED")
		}
		usageSeen = true
		if *u.Input <= step.Input && *u.Output <= step.Output {
			reply.Usage, reply.UsageSource = *u.Total, "actual"
		}
		return nil
	}
	consume := func(data string) error {
		if data == "[DONE]" {
			if finish == "" {
				return failure("MODEL_STREAM_INCOMPLETE")
			}
			done = true
			return nil
		}
		if !utf8.ValidString(data) {
			return failure("MODEL_PROTOCOL_ERROR")
		}
		var fields map[string]json.RawMessage
		if httpjson.DecodeObject([]byte(data), &fields) != nil {
			return failure("MODEL_PROTOCOL_ERROR")
		}
		var v chunk
		if json.Unmarshal([]byte(data), &v) != nil || v.ID == "" || len(v.ID) > 256 || len(v.Error) > 0 && string(v.Error) != "null" || len(v.Choices) > 1 {
			return failure("MODEL_PROTOCOL_ERROR")
		}
		if id == "" {
			id = v.ID
		}
		if v.ID != id {
			return failure("MODEL_PROTOCOL_ERROR")
		}
		if freeOnly && v.Usage != nil && len(v.Choices) == 1 && finish != "" {
			accounting := v.Choices[0]
			if accounting.Index == nil || *accounting.Index != 0 || accounting.Finish == nil || *accounting.Finish != finish || accounting.Delta.Content != "" || len(accounting.Delta.Tools) != 0 || accounting.Delta.Refusal != "" || (len(accounting.Delta.Legacy) > 0 && string(accounting.Delta.Legacy) != "null") || (accounting.Delta.Role != "" && accounting.Delta.Role != "assistant") {
				return failure("MODEL_USAGE_INVALID")
			}
			return recordUsage(v.Usage)
		}
		attachedUsage := freeOnly && v.Usage != nil && len(v.Choices) == 1 && finish == "" && v.Choices[0].Finish != nil
		if v.Usage != nil && !attachedUsage {
			if len(v.Choices) != 0 {
				return failure("MODEL_USAGE_INVALID")
			}
			if err := recordUsage(v.Usage); err != nil {
				return err
			}
		}
		if len(v.Choices) == 0 {
			if v.Usage == nil {
				return failure("MODEL_PROTOCOL_ERROR")
			}
			return nil
		}
		if finish != "" || usageSeen {
			return failure("MODEL_PROTOCOL_ERROR")
		}
		choice := v.Choices[0]
		if choice.Index == nil || *choice.Index != 0 || choice.Delta.Role != "" && choice.Delta.Role != "assistant" || choice.Delta.Refusal != "" || len(choice.Delta.Legacy) > 0 && string(choice.Delta.Legacy) != "null" {
			return failure("MODEL_PROTOCOL_ERROR")
		}
		text.WriteString(choice.Delta.Content)
		if text.Len() > 65536 {
			return failure("MODEL_RESPONSE_TOO_LARGE")
		}
		for _, t := range choice.Delta.Tools {
			if t.Index == nil || *t.Index != 0 || t.Type != "" && t.Type != "function" || len(t.ID) > 256 {
				return failure("MODEL_TOOL_UNSUPPORTED")
			}
			toolSeen = true
			if t.ID != "" {
				if callID != "" && t.ID != callID {
					return failure("MODEL_PROTOCOL_ERROR")
				}
				callID = t.ID
			}
			name.WriteString(t.Function.Name)
			args.WriteString(t.Function.Arguments)
			if name.Len() > 64 || args.Len() > 8192 {
				return failure("MODEL_RESPONSE_TOO_LARGE")
			}
		}
		if choice.Finish != nil {
			switch *choice.Finish {
			case "stop", "tool_calls", "length", "content_filter":
				finish = *choice.Finish
			default:
				return failure("MODEL_PROTOCOL_ERROR")
			}
		}
		if attachedUsage {
			return recordUsage(v.Usage)
		}
		return nil
	}
	for scanner.Scan() {
		if limited.N <= 0 {
			return execution.ModelReply{}, failure("MODEL_RESPONSE_TOO_LARGE")
		}
		line := scanner.Text()
		if line == "" {
			if event.Len() == 0 {
				continue
			}
			data := strings.TrimSuffix(event.String(), "\n")
			event.Reset()
			if err := consume(data); err != nil {
				return execution.ModelReply{}, err
			}
			if done {
				break
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, hasColon := strings.Cut(line, ":")
		if !hasColon {
			value = ""
		}
		value = strings.TrimPrefix(value, " ")
		if field == "data" {
			event.WriteString(value)
			event.WriteByte('\n')
			if event.Len() > 131072 {
				return execution.ModelReply{}, failure("MODEL_RESPONSE_TOO_LARGE")
			}
		}
	}
	if !done || scanner.Err() != nil {
		return execution.ModelReply{}, failure("MODEL_STREAM_INCOMPLETE")
	}
	if freeOnly && !usageSeen {
		return execution.ModelReply{}, failure("MODEL_COST_UNVERIFIED")
	}
	switch finish {
	case "stop":
		if toolSeen || strings.TrimSpace(text.String()) == "" {
			return execution.ModelReply{}, failure("MODEL_PROTOCOL_ERROR")
		}
		reply.Finish, reply.Text = "final", text.String()
	case "tool_calls":
		if !toolSeen || callID == "" || !agents.Alias(name.String()) {
			return execution.ModelReply{}, failure("MODEL_PROTOCOL_ERROR")
		}
		var command execution.ShellCommand
		if err := httpjson.DecodeObject([]byte(args.String()), &command); err != nil {
			return execution.ModelReply{}, failure("MODEL_TOOL_ARGUMENTS_INVALID")
		}
		// Preamble is not a final answer. The domain will validate the command and
		// classify the alias against pinned policy before any approval or dispatch.
		reply.Finish = "tool"
		reply.Tool = &execution.ShellProposal{Alias: name.String(), Command: command}
	default:
		reply.Finish = "error" // length/content_filter never becomes a final result.
	}
	return reply, nil
}
