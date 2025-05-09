package bot

import (
    "log"
    "os"
    "strings"

    "github.com/airylvat/trivia-bot/db"

    "github.com/bwmarrin/discordgo"
    "github.com/joho/godotenv"
)

type Bot struct {
    Session   *discordgo.Session
    DB        *db.DB
    Trivia    *Trivia
    AdminID   string
    AdminRoleID string
}

func (b *Bot) isAdmin(s *discordgo.Session, m *discordgo.MessageCreate) bool {
    // Check if the user is the hardcoded admin
    if m.Author.ID == b.AdminID {
        return true
    }

    // Check if the user has the admin role
    if b.AdminRoleID == "" {
        return false // No admin role configured
    }

    member, err := s.GuildMember(m.GuildID, m.Author.ID)
    if err != nil {
        log.Printf("Error fetching member roles: %v", err)
        return false
    }

    for _, roleID := range member.Roles {
        if roleID == b.AdminRoleID {
            return true
        }
    }

    return false
}

func NewBot() (*Bot, error) {
    if err := godotenv.Load(); err != nil {
        return nil, err
    }

    token := os.Getenv("DISCORD_TOKEN")
    adminID := os.Getenv("ADMIN_ID")
    adminRoleID := os.Getenv("ADMIN_ROLE_ID")

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
        AdminRoleID: adminRoleID,
    }

    session.AddHandler(bot.handleMessage)
    return bot, nil
}

func (b *Bot) Start() error {
    if err := b.Session.Open(); err != nil {
        return err
    }
    log.Println("Bot is running...")
    log.Printf("Logged in as: %s#%s\n", b.Session.State.User.Username, b.Session.State.User.Discriminator)
    log.Printf("Admin ID: %s\n", b.AdminID)
    log.Printf("Admin Role ID: %s\n", b.AdminRoleID)
    log.Printf("Allowed Channels: %s\n", os.Getenv("ALLOWED_CHANNELS"))
    return nil
}

func (b *Bot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
    if m.Author.ID == s.State.User.ID {
        return
    }

    allowedChannels := strings.Split(os.Getenv("ALLOWED_CHANNELS"), ",")
    if len(allowedChannels) == 0 || allowedChannels[0] == "" {
        // If no channels are specified, respond in all channels
        return
    }

    allowed := false
    for _, channelID := range allowedChannels {
        if m.ChannelID == strings.TrimSpace(channelID) {
            allowed = true
            break
        }
    }
    if !allowed {
        return
    }

    switch {
    case m.Content == "!!trivia start" && b.isAdmin(s, m):
        b.handleStart(s, m)
    case m.Content == "!!trivia help":
        b.handleHelp(s, m)
    case m.Content == "!!trivia scores":
        b.handleScores(s, m)
    case m.Content == "!!trivia end" && b.isAdmin(s, m):
        b.handleEnd(s, m)
    case m.Content == "!!trivia next" && b.isAdmin(s, m):
        b.handleNext(s, m)
    case m.Content == "!!trivia reset" && b.isAdmin(s, m):
        b.handleReset(s, m)
    case len(m.Content) > len("!!trivia join ") && m.Content[:13] == "!!trivia join":
        b.handleJoin(s, m)
    case len(m.Content) > len("!!trivia answer ") && m.Content[:15] == "!!trivia answer":
        b.handleAnswer(s, m)
    case len(m.Content) > len("!!trivia addq ") && m.Content[:13] == "!!trivia addq" && b.isAdmin(s, m):
        b.handleAddQuestion(s, m)
    case len(m.Content) > len("!!trivia removeq ") && m.Content[:16] == "!!trivia removeq" && b.isAdmin(s, m):
        b.handleRemoveQuestion(s, m)
    case m.Content == "!!trivia list" && b.isAdmin(s, m):
        b.handleListQuestions(s, m, "count")
    case m.Content == "!!trivia list answers" && b.isAdmin(s, m):
        b.handleListQuestions(s, m, "answers")
    case m.Content == "!!trivia list questions" && b.isAdmin(s, m):
        b.handleListQuestions(s, m, "count")
    }
}
