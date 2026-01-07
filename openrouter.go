package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"strings"
	"time"

	"github.com/mozillazg/go-unidecode"
	"github.com/revrost/go-openrouter"
	"github.com/revrost/go-openrouter/jsonschema"
)

type EchoTag struct {
	Categories []string `json:"categories,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Caption    string   `json:"caption,omitempty"`
	Text       []string `json:"text,omitempty"`
	Safety     string   `json:"safety,omitempty"`

	Similarity float32 `json:"similarity,omitempty"`
}

var (
	//go:embed prompt.txt
	TagPrompt string

	TagSchema = jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"categories": {
				Type:        jsonschema.Array,
				Description: "1-3 broad categories for filtering/search",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
					Enum: []string{
						"game",
						"desktop",
						"browser",
						"code",
						"terminal",
						"chat",
						"media",
						"document",
						"data",
						"map",
						"photo",
						"error",
						"other",
					},
				},
			},
			"tags": {
				Type:        jsonschema.Array,
				Description: "Tags describing the image",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"caption": {
				Type:        jsonschema.String,
				Description: "Caption describing the image",
			},
			"text": {
				Type:        jsonschema.Array,
				Description: "Any text visible in the image (ocr), if applicable",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"safety": {
				Type:        jsonschema.String,
				Description: "Safety of the image",
				Enum: []string{
					"ok",
					"sensitive",
				},
			},
		},
		Required: []string{
			"categories",
			"tags",
			"caption",
			"text",
			"safety",
		},
		AdditionalProperties: false,
	}
)

func (e *Echo) GenerateTags() {
	if config.AI.OpenRouterToken == "" || !e.IsImage() {
		return
	}

	log.Debugf("[%s] Tagging...\n", e.Hash)

	img, err := e.ReadAsJpegBase64()
	if err != nil {
		log.Warnf("[%s] Failed to read echo as jpeg: %v\n", e.Hash, err)

		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := openrouter.NewClient(config.AI.OpenRouterToken, openrouter.WithXTitle("Echo-Vault"), openrouter.WithHTTPReferer("https://github.com/coalaura/echo-vault"))

	request := openrouter.ChatCompletionRequest{
		Model:       config.AI.TaggingModel,
		Temperature: 0.3,
		MaxTokens:   256,
		Messages: []openrouter.ChatCompletionMessage{
			openrouter.SystemMessage(TagPrompt),
			openrouter.UserMessageWithImage("Analyze this image", img),
		},
		Provider: &openrouter.ChatProvider{
			DataCollection: openrouter.DataCollectionDeny,
			Sort:           openrouter.ProviderSortingPrice,
		},
		ResponseFormat: &openrouter.ChatCompletionResponseFormat{
			Type: openrouter.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openrouter.ChatCompletionResponseFormatJSONSchema{
				Name:   "result",
				Schema: &TagSchema,
				Strict: true,
			},
		},
	}

	completion, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		log.Warnf("[%s] Tag completion failed: %v\n", e.Hash, err)

		return
	}

	choices := completion.Choices

	if len(choices) == 0 {
		log.Warnf("[%s] Tag completion returned no choices\n", e.Hash)

		return
	}

	content := choices[0].Message.Content.Text

	if len(content) == 0 {
		log.Warnf("[%s] Tag completion returned no content", e.Hash)

		return
	}

	var result EchoTag

	err = json.Unmarshal([]byte(content), &result)
	if err != nil {
		log.Warnf("[%s] Tag completion returned bad json: %v\n", e.Hash, err)

		return
	}

	err = result.Clean()
	if err != nil {
		log.Warnf("[%s] Tag completion invalid: %v\n", e.Hash, err)

		return
	}

	err = vector.Store(e.Hash, result)
	if err != nil {
		log.Warnf("[%s] Tag completion invalid: %v\n", e.Hash, err)

		return
	}

	err = database.SetTags(e.Hash, result)
	if err != nil {
		log.Warnf("[%s] Failed to store tags: %v\n", e.Hash, err)

		return
	}

	log.Debugf("[%s] Tagging completed\n", e.Hash)
}

func (e *Echo) ReadAsJpegBase64() (string, error) {
	file, err := OpenFileForReading(e.Storage())
	if err != nil {
		return "", err
	}

	defer file.Close()

	img, err := decodeImage(file)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	err = jpeg.Encode(&buf, img, &jpeg.Options{
		Quality: 90,
	})

	if err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	return fmt.Sprintf("data:image/jpeg;base64,%s", b64), nil
}

func (t *EchoTag) Clean() error {
	if len(t.Categories) < 1 {
		return errors.New("missing categories")
	} else if len(t.Categories) > 3 {
		return fmt.Errorf("too many categories: %d", len(t.Categories))
	}

	if len(t.Tags) < 1 {
		return errors.New("missing tags")
	} else if len(t.Tags) > 32 {
		return fmt.Errorf("too many tags: %d", len(t.Tags))
	}

	t.Caption = strings.TrimSpace(t.Caption)

	if len(t.Caption) < 1 {
		return errors.New("missing caption")
	} else if len(t.Caption) > 256 {
		return fmt.Errorf("caption too long: %d", len(t.Caption))
	}

	if t.Safety != "ok" && t.Safety != "sensitive" {
		return fmt.Errorf("invalid safety tag: %q", t.Safety)
	}

	if len(t.Text) > 10 {
		return fmt.Errorf("too much text: %d", len(t.Text))
	}

	t.Caption = unidecode.Unidecode(t.Caption)

	for i, tag := range t.Tags {
		tag = strings.TrimSpace(tag)

		if len(tag) < 1 {
			return fmt.Errorf("tag %d empty", i+1)
		} else if len(tag) > 32 {
			return fmt.Errorf("tag %d too long: %d", i+1, len(tag))
		}

		t.Tags[i] = unidecode.Unidecode(tag)
	}

	for i, text := range t.Text {
		text = strings.TrimSpace(text)

		if len(text) < 1 {
			return fmt.Errorf("text %d empty", i+1)
		} else if len(text) > 128 {
			return fmt.Errorf("text %d too long: %d", i+1, len(text))
		}

		t.Text[i] = unidecode.Unidecode(text)
	}

	return nil
}

func (t *EchoTag) Serialize() (string, string, string, string, string) {
	categories := strings.Join(t.Categories, ",")
	tags := strings.Join(t.Tags, ",")
	text := strings.Join(t.Text, "\n")

	return string(categories), string(tags), t.Caption, string(text), t.Safety
}

func (t *EchoTag) EmbeddingString() string {
	cats := strings.Join(t.Categories, "\x1E")
	tags := strings.Join(t.Tags, "\x1E")
	text := strings.Join(t.Text, "\x1E")

	return cats + "\x1F" + tags + "\x1F" + t.Caption + "\x1F" + text
}
