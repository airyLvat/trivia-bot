package main

import (
    "log"
    "github.com/airylvat/trivia-bot/bot"
)

func main() {
    bot, err := bot.NewBot()
    if err != nil {
        log.Fatal(err)
    }

    if err := bot.Start(); err != nil {
        log.Fatal(err)
    }

    select {} // Keep bot running
}
