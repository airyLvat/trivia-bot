package bot

import (
    "github.com/bwmarrin/discordgo"
    "github.com/joho/godotenv"
    "log"
    "os"
    "github.com/airylvat/trivia-bot/db"
)

type Bot struct {
    Session   *discordgo.Session
    DB        *db.DB
    Trivia    *Trivia
    AdminID   string
}

func NewBot() (*Bot, error) {
    if err := godotenv.Load(); err != nil {
        return nil, err
    }

    token := os.Getenv("DISCORD_TOKEN")
    adminID := os.Getenv("ADMIN_ID")

    session, err := discordgo.New("Bot " + token)
    if err != nil {
        return nil, err
    }

    db, err := db.NewDB()
    if err != nil {
        return nil, err
    }

    bot := &Bot{
        Session: session,
        DB:      db,
        Trivia:  NewTrivia(),
        AdminID: adminID,
    }

    session.AddHandler(bot.handleMessage)
    return bot, nil
}

func (b *Bot) Start() error {
    if err := b.Session.Open(); err != nil {
        return err
    }
    log.Println("Bot is running...")
    return nil
}

func (b *Bot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
    if m.Author.ID == s.State.User.ID {
        return
    }

    switch {
    case m.Content == "!!trivia start":
        b.handleStart(s, m)
    case m.Content == "!!trivia help":
        b.handleHelp(s, m)
    case m.Content == "!!trivia scores":
        b.handleScores(s, m)
    case m.Content == "!!trivia end":
        b.handleEnd(s, m)
    case m.Content == "!!trivia next" && m.Author.ID == b.AdminID: // New command
        b.handleNext(s, m)
    case len(m.Content) > len("!!trivia join ") && m.Content[:13] == "!!trivia join":
        b.handleJoin(s, m)
    case len(m.Content) > len("!!trivia answer ") && m.Content[:15] == "!!trivia answer":
        b.handleAnswer(s, m)
    case len(m.Content) > len("!!trivia addq ") && m.Content[:13] == "!!trivia addq" && m.Author.ID == b.AdminID:
        b.handleAddQuestion(s, m)
    case len(m.Content) > len("!!trivia removeq ") && m.Content[:16] == "!!trivia removeq" && m.Author.ID == b.AdminID:
        b.handleRemoveQuestion(s, m)
    }
}
