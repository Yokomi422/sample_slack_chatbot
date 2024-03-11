package main

import (
	"github.com/Yokomi422/sample_slack_chatbot/gpt"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"github.com/slack-go/slack"
	"log"
	"os"
)

const (
	CHANNELNAME = "your_channel"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	SLACKCLIENTTOKEN := os.Getenv("SLACK_CLIENT_SECRET")
	OpenApiKey := os.Getenv("OPENAI_API_KEY")

	client := openai.NewClient(OpenApiKey)
	prompt := "sample_message"
	response, err := gpt.SendPrompt(client, prompt)
	if err != nil {
		log.Fatal(err)
	}

	c := slack.New(SLACKCLIENTTOKEN)
	_, _, er := c.PostMessage(CHANNELNAME, slack.MsgOptionText(response, true))
	if er != nil {
		panic(er)
	}
}
