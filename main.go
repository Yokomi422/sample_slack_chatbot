package main

import (
	"encoding/json"
	"fmt"
	"github.com/Yokomi422/sample_slack_chatbot/gpt"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	SLACKCLIENTTOKEN := os.Getenv("SLACK_CLIENT_SECRET")
	SLACKAPPTOKEN := os.Getenv("SLACK_APP_TOKEN")
	OpenApiKey := os.Getenv("OPENAI_API_KEY")

	client := openai.NewClient(OpenApiKey)
	api := slack.New(SLACKCLIENTTOKEN, slack.OptionAppLevelToken(SLACKAPPTOKEN))
	socketClient := socketmode.New(
		api,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	http.HandleFunc("/slack/events", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var payload map[string]interface{}
		err = json.Unmarshal(body, &payload)
		if err != nil {
			log.Printf("Error parsing request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if payload["type"] == "url_verification" {
			challenge := payload["challenge"].(string)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(challenge))
			return
		}

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: os.Getenv("SLACK_VERIFICATION_TOKEN")}))
		if err != nil {
			log.Printf("Error parsing Slack event: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				prompt := strings.TrimSpace(strings.TrimPrefix(ev.Text, fmt.Sprintf("<@%s>", ev.BotID)))
				response, err := gpt.SendPrompt(client, prompt)
				if err != nil {
					log.Printf("Error sending prompt to OpenAI: %v", err)
					return
				}
				_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(response, false))
				if err != nil {
					log.Printf("Error posting message to Slack: %v", err)
					return
				}
			}
		}
	})

	go func() {
		log.Println("[INFO] Server listening")
		http.ListenAndServe(":3000", nil)
	}()
	socketClient.Run()
}
