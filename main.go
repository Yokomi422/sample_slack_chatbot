package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Yokomi422/sample_slack_chatbot/gpt"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
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
				englishPrompt, err := gpt.TranslateToEnglish(client, prompt)
				if err != nil {
					log.Printf("Error translating prompt to English: %v", err)
					return
				}

				intent, err := gpt.DetectIntent(client, englishPrompt)
				if err != nil {
					log.Printf("Error detecting intent: %v", err)
					return
				}

				switch intent {
				case "programming":
					englishPrompt, err = gpt.ApplyProgrammingPromptTemplate(englishPrompt)
					if err != nil {
						log.Printf("Error applying programming prompt template: %v", err)
						return
					}
				case "general":
					englishPrompt, err = gpt.ApplyGeneralPromptTemplate(englishPrompt)
					if err != nil {
						log.Printf("Error applying general prompt template: %v", err)
						return
					}
				default:
					englishPrompt += " in Japanese"
				}

				response, err := gpt.SendPrompt(client, englishPrompt)
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
		log.Println("[INFO] Server listening on :3000")
		if err := http.ListenAndServe(":3000", nil); err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		for evt := range socketClient.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				fmt.Println("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				fmt.Println("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				fmt.Println("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					fmt.Printf("Ignored %+v\n", evt)
					continue
				}

				socketClient.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						prompt := strings.TrimSpace(strings.TrimPrefix(ev.Text, fmt.Sprintf("<@%s>", ev.BotID)))
						englishPrompt, err := gpt.TranslateToEnglish(client, prompt)
						if err != nil {
							log.Printf("Error translating prompt to English: %v", err)
							return
						}

						intent, err := gpt.DetectIntent(client, englishPrompt)
						if err != nil {
							log.Printf("Error detecting intent: %v", err)
							return
						}

						switch intent {
						case "programming":
							englishPrompt, err = gpt.ApplyProgrammingPromptTemplate(englishPrompt)
							if err != nil {
								log.Printf("Error applying programming prompt template: %v", err)
								return
							}
						case "general":
							englishPrompt += " Please provide a clear and concise answer in Japanese."
						default:
							englishPrompt += " in Japanese"
						}

						response, err := gpt.SendPrompt(client, englishPrompt)
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
				default:
					socketClient.Debugf("Unsupported Events API event received")
				}
			default:
				fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()

	socketClient.Run()
}
