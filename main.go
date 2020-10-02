package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/retailnext/unixtime"
)

var daysAgo = flag.Int("ago", 90, "Number of days back")
var apiToken = flag.String("api_token", "", "API token")
var cookie = flag.String("cookie", "", "Cookie (only for session tokens)")
var channelName = flag.String("channel", "", "Channel name")
var printAttachments = flag.Bool("attachments", false, "Print attachments")
var dumpJson = flag.Bool("json", false, "Dump a JSON stream")

func main() {
	flag.Parse()

	if *apiToken == "" {
		log.Fatal("-api_token required")
	}

	if *channelName == "" {
		log.Fatal("-channel is required")
	}

	var options []slack.Option
	if *cookie != "" {
		jar, err := cookiejar.New(nil)
		if err != nil {
			panic(err)
		}
		u, err := url.Parse("https://slack.com")
		if err != nil {
			panic(err)
		}

		fakeReq := fmt.Sprintf("GET / HTTP/1.0\r\nCookie: %s\r\n\r\n", *cookie)
		req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(fakeReq)))
		if err != nil {
			panic(err)
		}

		jar.SetCookies(u, req.Cookies())
		client := http.Client{
			Jar: jar,
		}
		options = append(options, slack.OptionHTTPClient(&client))
	}

	api := slack.New(*apiToken, options...)

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
		outJson := json.NewEncoder(os.Stdout)

		for _, msg := range history.Messages {
			ts, err := strconv.ParseFloat(msg.Timestamp, 64)
			if err != nil {
				log.Fatal(err)
			}

			t := unixtime.ToTime(int64(ts), time.Second)
			lastTime = t

			if *dumpJson {
				err := outJson.Encode(msg)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Printf("%s %-8.8s: %s\n", t.Format(time.RFC3339), msg.Username, msg.Text)
				if *printAttachments {
					for _, atmt := range msg.Attachments {
						fmt.Printf("atmt: %s\n", atmt.Fallback)
					}
				}
			}
		}
	}
}
