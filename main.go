package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/nlopes/slack"
	"github.com/retailnext/unixtime"
)

var daysAgo = flag.Int("ago", 90, "Number of days back")
var apiToken = flag.String("api_token", "", "API token")
var channelName = flag.String("channel", "", "Channel name")

func main() {
	flag.Parse()

	if *apiToken == "" {
		log.Fatal("-api_token required")
	}

	if *channelName == "" {
		log.Fatal("-channel is required")
	}

	api := slack.New(*apiToken)
	channels, err := api.GetChannels(false)
	if err != nil {
		log.Fatalf("GetChannels error: %s", err)
	}

	var chanID string
	for _, channel := range channels {
		if channel.Name == *channelName {
			chanID = channel.ID
			break
		}
	}

	if chanID == "" {
		log.Fatalf("No channel found for name %s", *channelName)
	}

	oldest := time.Now().AddDate(0, 0, -*daysAgo)

	lastTime := time.Now()

	for lastTime.After(oldest) {
		params := slack.HistoryParameters{
			Oldest: strconv.Itoa(int(oldest.Unix())),
			Latest: strconv.Itoa(int(lastTime.Unix())),
			Count:  1000,
		}
		history, err := api.GetChannelHistory(chanID, params)
		if err != nil {
			log.Fatalf("GetHistory error: %s", err)
		}

		if len(history.Messages) < 1 {
			break
		}

		for _, msg := range history.Messages {
			ts, err := strconv.ParseFloat(msg.Timestamp, 64)
			if err != nil {
				log.Fatal(err)
			}

			t := unixtime.ToTime(int64(ts), time.Second)
			lastTime = t

			fmt.Printf("%s %-8.8s: %s\n", t.Format(time.RFC3339), msg.Username, msg.Text)
		}
	}
}
