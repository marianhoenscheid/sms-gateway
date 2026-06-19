package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func env(name string) string {
	return os.Getenv(name)
}

func main() {
	token := env("TELEGRAM_TOKEN")
	chatID := env("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		fmt.Fprintln(os.Stderr, "sms-telegram: TELEGRAM_TOKEN and TELEGRAM_CHAT_ID required")
		os.Exit(1)
	}

	var text strings.Builder
	if parts, _ := strconv.Atoi(env("DECODED_PARTS")); parts > 0 {
		for i := 1; i <= parts; i++ {
			text.WriteString(env(fmt.Sprintf("DECODED_%d_TEXT", i)))
		}
	} else {
		messages, _ := strconv.Atoi(env("SMS_MESSAGES"))
		if messages == 0 {
			messages = 1
		}
		for i := 1; i <= messages; i++ {
			text.WriteString(env(fmt.Sprintf("SMS_%d_TEXT", i)))
		}
	}

	sender := env("SMS_1_NUMBER")
	if sender == "" {
		sender = "unbekannt"
	}
	msg := fmt.Sprintf("📱 SMS von %s:\n%s", sender, text.String())

	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", msg)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.PostForm(apiURL, form)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sms-telegram: request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
}
