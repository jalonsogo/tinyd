package dmr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tinyd/internal/types"
)

// ChatEndpoint returns the OpenAI-compatible chat completions endpoint.
// Exposed so the UI can surface it in the API info panel.
func (c *Client) ChatEndpoint() string {
	return c.baseURL + "/engines/llama.cpp/v1/chat/completions"
}

// ChatStreamResult bundles the streaming response so callers can read
// chunks one by one and close when done.
type ChatStreamResult struct {
	Reader *bufio.Reader
	Body   io.Closer
}

// ChatStream opens a streaming chat completion against DMR. The OpenAI-
// compatible endpoint returns SSE-style "data: {json}" lines that the
// caller drains one at a time via NextChatToken.
func (c *Client) ChatStream(ctx context.Context, modelRef string, messages []types.ChatMessage) (*ChatStreamResult, error) {
	type apiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	apiMsgs := make([]apiMsg, len(messages))
	for i, m := range messages {
		apiMsgs[i] = apiMsg{Role: m.Role, Content: m.Content}
	}
	payload, err := json.Marshal(map[string]interface{}{
		"model":    modelRef,
		"messages": apiMsgs,
		"stream":   true,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.ChatEndpoint(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, wrapErr(err, "chat stream")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("chat failed (%d): %s", resp.StatusCode, truncate(string(body), 200))
	}
	return &ChatStreamResult{
		Reader: bufio.NewReader(resp.Body),
		Body:   resp.Body,
	}, nil
}

// NextChatToken reads the next SSE event from r and returns the decoded
// content token. done is true at the end of the stream (after "[DONE]" or
// EOF). The caller is responsible for closing the underlying body when
// done becomes true or an error is returned.
func NextChatToken(r *bufio.Reader) (token string, done bool, err error) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return "", true, nil
			}
			return "", true, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue // SSE event separator
		}
		if !strings.HasPrefix(line, "data:") {
			continue // ignore comments and unknown lines
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return "", true, nil
		}
		// OpenAI-compatible chunk shape:
		//   {"choices":[{"delta":{"content":"hello"},"finish_reason":null}]}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if jerr := json.Unmarshal([]byte(payload), &chunk); jerr != nil {
			// Ignore malformed chunks rather than dying — DMR occasionally
			// emits heartbeat-like lines we don't need to parse.
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		if ch.Delta.Content != "" {
			return ch.Delta.Content, false, nil
		}
		if ch.FinishReason != nil {
			return "", true, nil
		}
		// Empty delta — keep reading.
	}
}
