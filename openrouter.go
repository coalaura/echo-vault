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

type EchoMeta struct {
	Description string `json:"description"`
	Phrases     string `json:"phrases"`
	Safety      string `json:"safety"`
}

var (
	//go:embed prompt.txt
	TagPrompt string

	TagSchema = jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"description": {
				Type:        jsonschema.String,
				Description: "Natural visual description (50-100 words)",
			},
			"phrases": {
				Type:        jsonschema.String,
				Description: "Comma-separated casual search phrases (3-5 phrases, 2-4 words each)",
			},
			"safety": {
				Type:        jsonschema.String,
				Description: "Content safety classification",
				Enum:        []string{"ok", "suggestive", "explicit", "violence", "selfharm", "sensitive"},
			},
		},
		Required:             []string{"description", "phrases", "safety"},
		AdditionalProperties: false,
	}
)

func (e *Echo) GenerateTags(ctx context.Context, noLogs, noSync bool) float64 {
	if config.AI.OpenRouterToken == "" || !e.IsImage() || e.Animated {
		return 0
	}

	if !noLogs {
		log.Debugf("[%s] Tagging...\n", e.Hash)
	}

	if !noSync {
		hub.BroadcastProcessing(e.Hash, true)

		defer hub.BroadcastProcessing(e.Hash, false)
	}

	img, err := e.ReadAsJpegBase64()
	if err != nil {
		log.Warnf("[%s] Failed to read echo as jpeg: %v\n", e.Hash, err)

		return 0
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client := openrouter.NewClient(
		config.AI.OpenRouterToken,
		openrouter.WithXTitle("Echo-Vault"),
		openrouter.WithHTTPReferer("https://github.com/coalaura/echo-vault"),
	)

	request := openrouter.ChatCompletionRequest{
		Model:       config.AI.TaggingModel,
		Temperature: 0.2,
		MaxTokens:   300,
		Messages: []openrouter.ChatCompletionMessage{
			openrouter.SystemMessage(TagPrompt),
			openrouter.UserMessageWithImage("Describe this image.", img),
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
		Usage: &openrouter.IncludeUsage{
			Include: true,
		},
	}

	completion, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		log.Warnf("[%s] Tag completion failed: %v\n", e.Hash, err)

		return 0
	}

	var cost float64

	if completion.Usage != nil {
		cost = completion.Usage.Cost
	}

	choices := completion.Choices

	if len(choices) == 0 {
		log.Warnf("[%s] Tag completion returned no choices\n", e.Hash)

		return cost
	}

	content := choices[0].Message.Content.Text

	if len(content) == 0 {
		log.Warnf("[%s] Tag completion returned no content", e.Hash)

		return cost
	}

	var result EchoMeta

	err = json.Unmarshal([]byte(content), &result)
	if err != nil {
		log.Warnf("[%s] Tag completion returned bad json: %v\n", e.Hash, err)

		return cost
	}

	err = result.Clean()
	if err != nil {
		log.Warnf("[%s] Tag completion invalid: %v\n", e.Hash, err)

		return cost
	}

	err = vector.Store(e.Hash, result)
	if err != nil {
		log.Warnf("[%s] Failed to store vector: %v\n", e.Hash, err)

		return cost
	}

	err = database.SetTags(e.Hash, result)
	if err != nil {
		log.Warnf("[%s] Failed to store tags: %v\n", e.Hash, err)

		return cost
	}

	e.Safety = result.Safety

	if !noLogs {
		log.Debugf("[%s] Tagging completed\n", e.Hash)
	}

	if !noSync {
		hub.BroadcastUpdate(e)
	}

	return cost
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

	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	if err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	return fmt.Sprintf("data:image/jpeg;base64,%s", b64), nil
}

func (t *EchoMeta) Clean() error {
	// Description validation
	t.Description = strings.TrimSpace(t.Description)
	t.Description = unidecode.Unidecode(t.Description)

	if len(t.Description) < 30 {
		return errors.New("description too short (min 30 chars)")
	}

	if len(t.Description) > 1000 {
		t.Description = t.Description[:1000]
	}

	// Safety validation
	if !IsValidSafety(t.Safety) {
		return fmt.Errorf("invalid safety tag: %q", t.Safety)
	}

	return nil
}

func (t *EchoMeta) Serialize() (string, string, string) {
	return t.Description, t.Phrases, t.Safety
}

func (t *EchoMeta) Embedding() string {
	return t.Phrases + "\n\n" + t.Description
}

func IsValidSafety(safety string) bool {
	switch safety {
	case "ok", "suggestive", "explicit", "violence", "selfharm", "sensitive":
		return true
	}

	return false
}
