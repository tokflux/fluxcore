package message

import (
	"encoding/json"
	"testing"
)

func TestMessageMarshalJSON(t *testing.T) {
	t.Run("text_only_content", func(t *testing.T) {
		msg := Message{
			Role:    "assistant",
			Content: []Content{TextContent("Hello")},
		}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		content, ok := result["content"].(string)
		if !ok {
			t.Fatalf("expected content to be string, got %T: %v", result["content"], result["content"])
		}
		if content != "Hello" {
			t.Errorf("expected content 'Hello', got '%s'", content)
		}
		if result["role"] != "assistant" {
			t.Errorf("expected role 'assistant', got '%v'", result["role"])
		}
	})

	t.Run("empty_content", func(t *testing.T) {
		msg := Message{
			Role:    "assistant",
			Content: []Content{},
		}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		content, ok := result["content"].(string)
		if !ok {
			t.Fatalf("expected content to be string, got %T", result["content"])
		}
		if content != "" {
			t.Errorf("expected empty content, got '%s'", content)
		}
	})

	t.Run("multiple_text_content", func(t *testing.T) {
		msg := Message{
			Role: "assistant",
			Content: []Content{
				TextContent("Hello "),
				TextContent("World"),
			},
		}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		content, ok := result["content"].(string)
		if !ok {
			t.Fatalf("expected content to be string, got %T", result["content"])
		}
		if content != "Hello World" {
			t.Errorf("expected 'Hello World', got '%s'", content)
		}
	})
}

func TestMessageMarshalJSON_Streaming(t *testing.T) {
	t.Run("stream_chunk_text_only", func(t *testing.T) {
		chunk := StreamChunk{
			ID:     "chatcmpl-123",
			Object: "chat.completion.chunk",
			Model:  "gpt-4",
			Choices: []StreamChoice{{
				Index: 0,
				Delta: Message{
					Content: []Content{TextContent("Hello")},
				},
			}},
		}
		data, err := json.Marshal(chunk)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		choices := result["choices"].([]interface{})
		delta := choices[0].(map[string]interface{})["delta"].(map[string]interface{})
		content, ok := delta["content"].(string)
		if !ok {
			t.Fatalf("expected delta.content to be string, got %T: %v", delta["content"], delta["content"])
		}
		if content != "Hello" {
			t.Errorf("expected 'Hello', got '%s'", content)
		}
	})
}
