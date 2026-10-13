package modelgateway

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"portway/internal/execution"
	"portway/internal/problem"
)

const ProbeProfile = "chat-completions-v1"
const ProbeTimeout = 12 * time.Second
const ProbeOutputTokens = 128

// ProbeResult attests only an observed wire protocol. It does not prove a
// tokenizer, context window, future availability, or authority to dispatch.
type ProbeResult struct {
	Streaming     bool   `json:"streaming"`
	ToolCalling   bool   `json:"tool_calling"`
	ToolResult    bool   `json:"tool_result"`
	ReportedUsage bool   `json:"reported_usage"`
	Requests      int    `json:"requests"`
	ErrorCode     string `json:"error_code,omitempty"`
}

func (p ProbeResult) Compatible() bool {
	return p.Streaming && p.ToolCalling && p.ToolResult && p.ReportedUsage && p.ErrorCode == "" && p.Requests == 2
}

// Probe performs at most two bounded generations with fixed, nonsecret inputs.
// The proposed command is inspected as data and NEVER executed. A synthetic
// observation verifies tool-result roundtripping, independently of Runtime.
// max_tokens is requested, not a guarantee that an untrusted server obeys it.
func (c *Client) Probe(ctx context.Context, beforeRequest func(context.Context) error) ProbeResult {
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()
	result := ProbeResult{}
	tools := []toolDefinition{{Type: "function", Function: functionDefinition{
		Name: "portway_probe", Description: "Protocol test only; propose the requested arguments. Nothing will be executed.", Parameters: commandSchema,
	}}}
	messages := []message{{Role: "system", Content: "This is a protocol compatibility test. Follow the requested format exactly."},
		{Role: "user", Content: `Call portway_probe with {"argv":["echo","PORTWAY_PROBE_OK"],"working_dir":".","timeout_seconds":1}. This is a simulated tool; no command will run.`}}
	call := func(choice any) (execution.ModelReply, error) {
		body, _ := json.Marshal(map[string]any{"model": c.model, "messages": messages, "tools": tools,
			"tool_choice": choice, "parallel_tool_calls": false, "stream": true,
			"stream_options": map[string]bool{"include_usage": true}, "max_tokens": ProbeOutputTokens, "n": 1})
		if ctx.Err() != nil {
			return execution.ModelReply{}, failure("MODEL_CANCELLED")
		}
		if beforeRequest != nil {
			if err := beforeRequest(ctx); err != nil {
				return execution.ModelReply{}, err
			}
		}
		result.Requests++
		// Input is a parser ceiling, NOT a prompt reservation/token estimate.
		return c.completeBody(ctx, body, execution.Run{}, execution.Step{Input: 40000, Output: ProbeOutputTokens})
	}
	reply, err := call(map[string]any{"type": "function", "function": map[string]string{"name": "portway_probe"}})
	if err != nil {
		result.ErrorCode = problem.Public(err).Code
		return result
	}
	result.Streaming = true
	if reply.Finish != "tool" || reply.Tool == nil || reply.Tool.Alias != "portway_probe" {
		result.ErrorCode = "MODEL_PROBE_TOOL_MISMATCH"
		return result
	}
	command := reply.Tool.Command
	if len(command.Argv) != 2 || command.Argv[0] != "echo" || command.Argv[1] != "PORTWAY_PROBE_OK" || command.WorkingDir != "." || command.TimeoutSeconds != 1 {
		result.ErrorCode = "MODEL_PROBE_TOOL_MISMATCH"
		return result
	}
	result.ToolCalling = true
	usage := reply.UsageSource == "actual"
	args, _ := json.Marshal(command)
	messages = append(messages, message{Role: "assistant", ToolCalls: []toolCall{{ID: "call_portway_probe", Type: "function", Function: functionCall{Name: "portway_probe", Arguments: string(args)}}}},
		message{Role: "tool", ToolCallID: "call_portway_probe", Content: `{"simulated":true,"output":"PORTWAY_PROBE_OK"}`},
		message{Role: "user", Content: "Reply with exactly the output from the simulated tool, without any other text."})
	reply, err = call("none")
	if err != nil {
		result.ErrorCode = problem.Public(err).Code
		return result
	}
	if reply.Finish != "final" || strings.TrimSpace(reply.Text) != "PORTWAY_PROBE_OK" {
		result.ErrorCode = "MODEL_PROBE_RESULT_MISMATCH"
		return result
	}
	result.ToolResult = true
	result.ReportedUsage = usage && reply.UsageSource == "actual"
	if !result.ReportedUsage {
		result.ErrorCode = "MODEL_USAGE_UNAVAILABLE"
	}
	return result
}
