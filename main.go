package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func handleEventMessage(event *slackevents.EventsAPIEvent) error {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			// The application has been mentioned since this Event is a Mention event
			log.Println(ev)
		}
	default:
		return errors.New("unsupported event type")
	}
	return nil
}

func main() {
	// Load Env variables from .env file
	godotenv.Load(".env")

	token := os.Getenv("SLACK_AUTH_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")
	channelID := os.Getenv("SLACK_CHANNEL_ID")

	// Create new client to slack
	client := slack.New(
		token,
		slack.OptionDebug(true),
		slack.OptionAppLevelToken(appToken),
	)

	socketClient := socketmode.New(
		client,
		socketmode.OptionDebug(true),
		// Option to set a custom logger
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)
	// Create a context for the go routine
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	// Anon function to cancel context or attend to incoming events
	go func(ctx context.Context, client *slack.Client, socketClient *socketmode.Client) {
		for {
			select {
			case <-ctx.Done():
				log.Println("Shutting down socketmode listener")
				return
			case event := <-socketClient.Events:
				// If a new event has been passed interface{}
				switch event.Type {
				// handle EventAPi events
				case socketmode.EventTypeEventsAPI:

					eventsAPIEvent, ok := event.Data.(slackevents.EventsAPIEvent)
					if !ok {
						log.Printf("Could not type cast the event to the EventsAPIEvent: %v\n", event)
						continue
					}
					socketClient.Ack(*event.Request)

					err := handleEventMessage(&eventsAPIEvent)
					if err != nil {
						// Replace with actual err handeling
						log.Fatal(err)
					}
				}
			}
		}
	}(ctx, client, socketClient)

	err := socketClient.Run()

	if err != nil {
		panic(err)
	}

	// Create slack attachment
	attachment := slack.Attachment{
		Pretext: "Kunly Bot Message",
		Text:    "test text",
		Color:   "#36a64f",
		Fields: []slack.AttachmentField{
			{
				Title: "Date",
				Value: time.Now().String(),
			},
		},
	}

	// Post message to slack channel

	_, timestamp, err := client.PostMessage(
		channelID,

		slack.MsgOptionText("New message from bot", false),
		slack.MsgOptionAttachments(attachment),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Message sent at %s", timestamp)
}
