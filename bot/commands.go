package bot

import (
    "fmt"
    "log"
    "github.com/bwmarrin/discordgo"
    "strconv"
    "strings"
    "time"
)

func (b *Bot) handleStart(s *discordgo.Session, m *discordgo.MessageCreate) {
    if b.Trivia.Active {
        s.ChannelMessageSend(m.ChannelID, "Trivia is already running!")
        return
    }

    b.Trivia.Start()
    s.ChannelMessageSend(m.ChannelID, "Trivia started! Use `!!trivia join <team>` to join a team. Admin, use `!!trivia next` to post the first question.")

    go b.runTrivia(s, m.ChannelID)

    b.Trivia.NextChan <- struct{}{}
}

func (b *Bot) runTrivia(s *discordgo.Session, channelID string) {
    for b.Trivia.Active {
        // Wait for admin to trigger the next question
        select {
        case <-b.Trivia.NextChan:
        case <-time.After(5 * time.Minute):
            s.ChannelMessageSend(channelID, "Trivia timed out due to inactivity. Ending game.")
            b.Trivia.End()
            return
        }

        q, err := b.DB.GetRandomQuestion()
        if err != nil {
            s.ChannelMessageSend(channelID, "Error fetching question. Ending trivia.")
            b.Trivia.End()
            return
        }

        b.Trivia.SetQuestion(q)
        questionText := strings.TrimSpace(q.Text)
        log.Printf("Posting question: %q", questionText)
        s.ChannelMessageSend(channelID, fmt.Sprintf("Question: %s\nUse `!!trivia answer <answer>` to respond.", questionText))
    }
}

func (b *Bot) handleJoin(s *discordgo.Session, m *discordgo.MessageCreate) {
    team := strings.TrimSpace(m.Content[13:])
    if team == "" {
        s.ChannelMessageSend(m.ChannelID, "Please specify a team name.")
        return
    }

    if err := b.DB.JoinTeam(m.Author.ID, team); err != nil {
        s.ChannelMessageSend(m.ChannelID, "Error joining team.")
        return
    }

    s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s joined team %s!", m.Author.Username, team))
}

func (b *Bot) handleAnswer(s *discordgo.Session, m *discordgo.MessageCreate) {
    if !b.Trivia.Active || b.Trivia.Current == nil {
        s.ChannelMessageSend(m.ChannelID, "No active trivia question.")
        return
    }

    answer := strings.TrimSpace(m.Content[15:]) // For !!trivia answer
    player, err := b.DB.Query("SELECT team FROM players WHERE user_id = ?", m.Author.ID)
    if err != nil || !player.Next() {
        s.ChannelMessageSend(m.ChannelID, "You must join a team first with `!!trivia join <team>`.")
        return
    }
    var team string
    player.Scan(&team)
    player.Close()

    team = strings.TrimSpace(team)
    log.Printf("Comparing answer: user=%q, correct=%q, team=%q", answer, b.Trivia.Current.Answer, team)
    if strings.EqualFold(answer, strings.TrimSpace(b.Trivia.Current.Answer)) { // Case-insensitive, trim both
        if err := b.DB.AddScore(m.Author.ID, team, 10); err != nil {
            s.ChannelMessageSend(m.ChannelID, "Error updating score.")
            return
        }
        s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s answered correctly for team %s! +10 points!", m.Author.Username, team))
    } else {
        s.ChannelMessageSend(m.ChannelID, "Incorrect answer.")
    }
}

func (b *Bot) handleAddQuestion(s *discordgo.Session, m *discordgo.MessageCreate) {
    parts := strings.SplitN(m.Content[13:], "|", 2)
    if len(parts) != 2 {
        s.ChannelMessageSend(m.ChannelID, "Usage: `!!trivia addq <question> | <answer>`")
        return
    }

    question, answer := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
    if err := b.DB.AddQuestion(question, answer); err != nil {
        s.ChannelMessageSend(m.ChannelID, "Error adding question.")
        return
    }

    s.ChannelMessageSend(m.ChannelID, "Question added successfully!")
}

func (b *Bot) handleRemoveQuestion(s *discordgo.Session, m *discordgo.MessageCreate) {
    id, err := strconv.Atoi(strings.TrimSpace(m.Content[16:]))
    if err != nil {
        s.ChannelMessageSend(m.ChannelID, "Invalid question ID.")
        return
    }

    if err := b.DB.RemoveQuestion(id); err != nil {
        s.ChannelMessageSend(m.ChannelID, "Error removing question.")
        return
    }

    s.ChannelMessageSend(m.ChannelID, "Question removed successfully!")
}

func (b *Bot) handleScores(s *discordgo.Session, m *discordgo.MessageCreate) {
    players, teams, err := b.DB.GetScores()
    if err != nil {
        s.ChannelMessageSend(m.ChannelID, "Error fetching scores.")
        return
    }

    var response strings.Builder
    response.WriteString("**Scores**\n\n**Players**\n")
    for _, p := range players {
        response.WriteString(fmt.Sprintf("<@%s> (Team %s): %d\n", p.UserID, p.Team, p.Score))
    }
    response.WriteString("\n**Teams**\n")
    for _, t := range teams {
        response.WriteString(fmt.Sprintf("%s: %d\n", t.Name, t.Score))
    }

    s.ChannelMessageSend(m.ChannelID, response.String())
}

func (b *Bot) handleEnd(s *discordgo.Session, m *discordgo.MessageCreate) {
    if !b.Trivia.Active {
        s.ChannelMessageSend(m.ChannelID, "No active trivia game.")
        return
    }

    b.Trivia.End()
    s.ChannelMessageSend(m.ChannelID, "Trivia ended! Use `!!trivia scores` to see results.")
}

func (b *Bot) handleHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
    lines := []string{
        "**Trivia Bot Help**",
        "Here are the available commands:",
        "",
        "- **!!trivia start**: Start a new trivia contest.",
        "- **!!trivia help**: Show this help message.",
        "- **!!trivia join <team>**: Join a team (e.g., `!!trivia join Red`). **Changing teams resets your score!**",
        "- **!!trivia answer <answer>**: Submit an answer to the current question (case-insensitive, e.g., `France` or `france`).",
        "- **!!trivia scores**: Display individual and team scores.",
        "- **!!trivia end**: End the current trivia contest.",
        "- **!!trivia next**: [Admin] Trigger the next question.",
        "- **!!trivia addq <question> | <answer>**: [Admin] Add a new question (e.g., `!!trivia addq What is 2+2? | 4`).",
        "- **!!trivia removeq <id>**: [Admin] Remove a question by ID.",
        "",
        "Note: Admin commands are restricted to the bot's admin user.",
    }
    helpMessage := strings.Join(lines, "\n")

    s.ChannelMessageSend(m.ChannelID, helpMessage)
}

func (b *Bot) handleNext(s *discordgo.Session, m *discordgo.MessageCreate) {
    if !b.Trivia.Active {
        s.ChannelMessageSend(m.ChannelID, "No active trivia game. Use `!!trivia start` to begin.")
        return
    }

    // Signal the next question
    select {
    case b.Trivia.NextChan <- struct{}{}:
        // Success, question will be posted by runTrivia
    default:
        s.ChannelMessageSend(m.ChannelID, "A question is already being posted. Please wait.")
    }
}
