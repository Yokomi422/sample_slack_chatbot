package gpt

import (
	"context"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
	"io/ioutil"
	"strings"
)

func SendPrompt(client *openai.Client, prompt string) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("ChatCompletion error: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func TranslateToEnglish(client *openai.Client, prompt string) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a translation assistant. Translate the given text to English.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("Translation error: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func DetectIntent(client *openai.Client, prompt string) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI assistant that detects the intent of a given prompt. Categorize the intent as either 'programming' or 'general'.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("Intent detection error: %v", err)
	}

	intent := resp.Choices[0].Message.Content
	return strings.ToLower(strings.TrimSpace(intent)), nil
}

func ApplyProgrammingPromptTemplate(query string) (string, error) {
	templateBytes, err := ioutil.ReadFile("templates/programming_template.txt")
	if err != nil {
		return "", fmt.Errorf("Error reading programming prompt template: %v", err)
	}

	template := string(templateBytes)
	prompt := strings.Replace(template, "${query}", query, -1)

	return prompt, nil
}
