package message

import (
	"encoding/json"
	"strings"
)

// ContentData is a sealed interface for content data types.
// Only TextData and MediaData implement this interface.
type ContentData interface {
	isContentData()
}

// TextData represents text content.
type TextData string

func (TextData) isContentData() {}

// MediaData represents media content (image, audio, video).
type MediaData struct {
	URL       string `json:"url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Base64    string `json:"base64,omitempty"`
}

func (MediaData) isContentData() {}

// Content represents multimodal content
type Content struct {
	Type string      `json:"type"` // text, image, audio
	Data ContentData `json:"-"`
}

// MarshalJSON implements custom JSON marshaling for Content.
// Uses OpenAI-compatible field names: "text" for text type, "image_url" for images.
func (c Content) MarshalJSON() ([]byte, error) {
	switch d := c.Data.(type) {
	case TextData:
		return json.Marshal(map[string]interface{}{
			"type": c.Type,
			"text": string(d),
		})
	case MediaData:
		key := "image_url"
		if c.Type == "audio" {
			key = "input_audio"
		}
		return json.Marshal(map[string]interface{}{
			"type": c.Type,
			key: map[string]interface{}{
				"url":    d.URL,
				"detail": d.MediaType,
			},
		})
	}
	return json.Marshal(map[string]interface{}{
		"type": c.Type,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Content.
// Accepts both OpenAI wire format ("text"/"image_url") and legacy IR format ("data").
func (c *Content) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data,omitempty"`
		Text json.RawMessage `json:"text,omitempty"`
		URL  json.RawMessage `json:"image_url,omitempty"`
		Aud  json.RawMessage `json:"input_audio,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Type = raw.Type
	switch raw.Type {
	case "text":
		// Try "text" field first (OpenAI format), then "data" (legacy)
		field := raw.Text
		if field == nil {
			field = raw.Data
		}
		var text string
		if err := json.Unmarshal(field, &text); err != nil {
			return err
		}
		c.Data = TextData(text)
	case "image", "audio":
		// Try OpenAI format fields first, then legacy "data"
		field := raw.URL
		if field == nil {
			field = raw.Aud
		}
		if field == nil {
			field = raw.Data
		}
		var media MediaData
		if err := json.Unmarshal(field, &media); err != nil {
			return err
		}
		c.Data = media
	}
	return nil
}

// TextContent creates text content
func TextContent(text string) Content {
	return Content{
		Type: "text",
		Data: TextData(text),
	}
}

// AsText returns the text if this is TextData, otherwise empty string.
func (c Content) AsText() string {
	if td, ok := c.Data.(TextData); ok {
		return string(td)
	}
	return ""
}

// IsText returns true if this is text content.
func (c Content) IsText() bool {
	return c.Type == "text"
}

// ExtractAllText concatenates all text from content items.
func ExtractAllText(contents []Content) string {
	var sb strings.Builder
	for _, c := range contents {
		if c.IsText() {
			sb.WriteString(c.AsText())
		}
	}
	return sb.String()
}
