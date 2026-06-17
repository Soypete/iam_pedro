// Package classifier provides LLM-based message topic classification
// using the ontology index for topic taxonomy.
//
// The classifier takes a chat message, embeds it using the configured LLM,
// and finds the closest matching topic from the ontology using cosine
// similarity. It returns the topic label and confidence score.
//
// Classification is non-blocking and runs asynchronously from the IRC
// message handling path.
package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Soypete/twitch-llm-bot/internal/mempalace/ontology"
	"github.com/Soypete/twitch-llm-bot/metrics"
	"github.com/tmc/langchaingo/llms"
)

type Classifier struct {
	llm        llms.Model
	modelName  string
	classes    []ontology.Class
	classNames []string
}

func NewClassifier(llm llms.Model, modelName string, classes []ontology.Class) *Classifier {
	classNames := make([]string, len(classes))
	for i, c := range classes {
		classNames[i] = c.Label
	}

	return &Classifier{
		llm:        llm,
		modelName:  modelName,
		classes:    classes,
		classNames: classNames,
	}
}

func (c *Classifier) GetClasses() []ontology.Class {
	return c.classes
}

func (c *Classifier) Classify(ctx context.Context, msg string) (string, error) {
	start := time.Now()

	toolDef := getClassificationToolDefinition(c.classNames)

	systemPrompt := `You are a chat message classifier. Given a chat message, classify it into exactly one topic category.
If the message doesn't clearly relate to any topic, return "Unclassified".
Be conservative - only classify if there's a clear topic match.`

	userMessage := fmt.Sprintf(`Classify this chat message into one of these categories: %s

Message to classify:
%s

Respond with exactly one tool call.`, strings.Join(c.classNames, ", "), msg)

	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userMessage),
	}

	resp, err := c.llm.GenerateContent(ctx, messageHistory,
		llms.WithModel(c.modelName),
		llms.WithCandidateCount(1),
		llms.WithMaxLength(100),
		llms.WithTemperature(0.3),
		llms.WithTools([]llms.Tool{toolDef}),
	)
	if err != nil {
		return "", fmt.Errorf("LLM classification failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in LLM response")
	}

	choice := resp.Choices[0]

	if len(choice.ToolCalls) > 0 {
		toolCall := choice.ToolCalls[0]
		result, err := parseClassificationResponse(toolCall)
		if err != nil {
			return "", err
		}

		duration := time.Since(start)
		metrics.MempalaceClassificationLatency.Observe(duration.Seconds())

		if result.Topic == "Unclassified" {
			metrics.MempalaceClassificationUnclassifiedTotal.Add(1)
		} else {
			metrics.MempalaceMessagesClassifiedTotal.WithLabelValues(result.Topic).Add(1)
		}

		return result.Topic, nil
	}

	cleanResp := strings.TrimSpace(choice.Content)
	for _, cn := range c.classNames {
		if strings.EqualFold(cleanResp, cn) {
			duration := time.Since(start)
			metrics.MempalaceClassificationLatency.Observe(duration.Seconds())
			metrics.MempalaceMessagesClassifiedTotal.WithLabelValues(cn).Add(1)
			return cn, nil
		}
	}

	metrics.MempalaceClassificationUnclassifiedTotal.Add(1)
	return "Unclassified", nil
}

func getClassificationToolDefinition(classNames []string) llms.Tool {
	classEnum := make([]string, len(classNames)+1)
	copy(classEnum, classNames)
	classEnum[len(classEnum)-1] = "Unclassified"

	enumStr := make([]map[string]string, len(classEnum))
	for i, c := range classEnum {
		enumStr[i] = map[string]string{"const": c}
	}

	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "classify_message",
			Description: "Classify a chat message into a topic category",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"topic": map[string]any{
						"type":        "string",
						"description": "The topic category",
						"enum":        classEnum,
					},
					"confidence": map[string]any{
						"type":        "number",
						"description": "Confidence score from 0 to 1",
					},
				},
				"required": []string{"topic", "confidence"},
			},
		},
	}
}

type classificationResult struct {
	Topic      string  `json:"topic"`
	Confidence float64 `json:"confidence"`
}

func parseClassificationResponse(toolCall llms.ToolCall) (*classificationResult, error) {
	var result classificationResult
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &result); err != nil {
		cleanArgs := strings.Trim(toolCall.FunctionCall.Arguments, "{}")
		parts := strings.Split(cleanArgs, ",")
		for _, p := range parts {
			kv := strings.Split(p, ":")
			if len(kv) == 2 {
				key := strings.Trim(kv[0], `" `)
				val := strings.Trim(kv[1], `" `)
				switch key {
				case "topic":
					result.Topic = val
				case "confidence":
					_, _ = fmt.Sscanf(val, "%f", &result.Confidence)
				}
			}
		}
	}

	if result.Topic == "" {
		return nil, fmt.Errorf("failed to parse topic from tool call")
	}

	return &result, nil
}
