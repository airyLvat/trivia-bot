package bot

import (
    "fmt"
    "log"
    "github.com/bwmarrin/discordgo"
    "strconv"
    "strings"
    "time"
)

const (
    timeoutLength = 10 // Minutes
)
func (b *Bot) handleStart(s *discordgo.Session, m *discordgo.MessageCreate) {
    if b.Trivia.Active {
        s.ChannelMessageSendReply(m.ChannelID, "Trivia is already running!", m.Reference())
        return
    }

    b.Trivia.Start()
    s.ChannelMessageSend(m.ChannelID, "Trivia started! Use `!!trivia join <team>` to join a team. Admin, use `!!trivia next` to post the first question. Use `!!trivia help` for more commands.")
    log.Printf("Trivia started by %s\n", m.Author.Username)

    go b.runTrivia(s, m.ChannelID)

    b.Trivia.NextChan <- struct{}{}
}

func (b *Bot) runTrivia(s *discordgo.Session, channelID string) {
    for b.Trivia.Active {
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
        questionNumber := b.Trivia.Current.ID
        log.Printf("Posting question: %q - %q", questionNumber, questionText)

        embed := &discordgo.MessageEmbed{
            Title:       "Trivia Question # " + strconv.Itoa(questionNumber),
            Description: questionText,
            Color:       0x00ff00, // Green sidebar
            Footer: &discordgo.MessageEmbedFooter{
                Text: "Use !!trivia answer <answer> to respond (case-insensitive). Only the first correct answer earns points.",
            },
        }

        _, err = s.ChannelMessageSendEmbed(channelID, embed)
        if err != nil {
            s.ChannelMessageSend(channelID, "Error posting question. Ending trivia.")
            log.Printf("Embed error: %v", err)
            b.Trivia.End()
            return
        }
    }
}

func (b *Bot) handleJoin(s *discordgo.Session, m *discordgo.MessageCreate) {
    team := strings.ToLower(strings.TrimSpace(m.Content[13:]))
    if team == "" {
        s.ChannelMessageSendReply(m.ChannelID, "Please specify a team name.", m.Reference())
        return
    }

    if err := b.DB.JoinTeam(m.Author.ID, team); err != nil {
        s.ChannelMessageSendReply(m.ChannelID, "Error joining team.", m.Reference())
        return
    }

    s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("%s joined team %s!", m.Author.Username, team), m.Reference())
    log.Printf("User %s joined team %s\n", m.Author.Username, team)
}

func (b *Bot) handleAnswer(s *discordgo.Session, m *discordgo.MessageCreate) {
    if !b.Trivia.Active || b.Trivia.Current == nil {
        s.ChannelMessageSendReply(m.ChannelID, "No active trivia question.", m.Reference())
        return
    }

    b.Trivia.Mutex.Lock()
    if b.Trivia.AnsweredCorrect {
        b.Trivia.Mutex.Unlock()
        s.ChannelMessageSendReply(m.ChannelID, "This question has already been answered correctly. Wait for the next question.", m.Reference())
        return
    }
    b.Trivia.Mutex.Unlock()

    answer := strings.TrimSpace(m.Content[15:])
    player, err := b.DB.Query("SELECT team FROM players WHERE user_id = ?", m.Author.ID)
    if err != nil || !player.Next() {
        s.ChannelMessageSendReply(m.ChannelID, "You must join a team first with `!!trivia join <team>`.", m.Reference())
        return
    }
    var team string
    player.Scan(&team)
    player.Close()

    team = strings.TrimSpace(team)
    log.Printf("Comparing answer: user=%q, correct=%q, team=%q", answer, b.Trivia.Current.Answer, team)
    if strings.EqualFold(answer, strings.TrimSpace(b.Trivia.Current.Answer)) {
        b.Trivia.Mutex.Lock()
        if b.Trivia.AnsweredCorrect { // Double-check in case of race
            b.Trivia.Mutex.Unlock()
            s.ChannelMessageSendReply(m.ChannelID, "This question has already been answered correctly. Wait for the next question.", m.Reference())
            return
        }
        b.Trivia.AnsweredCorrect = true
        b.Trivia.Mutex.Unlock()

        if err := b.DB.AddScore(m.Author.ID, team, 10); err != nil {
            s.ChannelMessageSendReply(m.ChannelID, "Error updating score.", m.Reference())
            log.Printf("Score update error: %v", err)
            return
        }
        s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("%s answered correctly for team %s! +10 points! Question closed, admin use `!!trivia next` for the next question.", m.Author.Username, team), m.Reference())
    } else {
        s.ChannelMessageSendReply(m.ChannelID, "Incorrect answer.", m.Reference())
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
        s.ChannelMessageSendReply(m.ChannelID, "Error adding question.", m.Reference())
        log.Println("Error adding question:", err)
        return
    }

    s.ChannelMessageSendReply(m.ChannelID, "Question added successfully!", m.Reference())
    log.Printf("Question added by %s: %q | %q\n", m.Author.Username, question, answer)
}

func (b *Bot) handleRemoveQuestion(s *discordgo.Session, m *discordgo.MessageCreate) {
    id, err := strconv.Atoi(strings.TrimSpace(m.Content[16:]))
    if err != nil {
        s.ChannelMessageSendReply(m.ChannelID, "Invalid question ID.", m.Reference())
        return
    }

    if err := b.DB.RemoveQuestion(id); err != nil {
        s.ChannelMessageSendReply(m.ChannelID, "Error removing question.", m.Reference())
        log.Println("Error removing question:", err)
        return
    }

    s.ChannelMessageSendReply(m.ChannelID, "Question removed successfully!", m.Reference())
    log.Printf("Question %d removed by %s\n", id, m.Author.Username)
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
    log.Printf("Scores requested by %s\n", m.Author.Username)
}

func (b *Bot) handleEnd(s *discordgo.Session, m *discordgo.MessageCreate) {
    if !b.Trivia.Active {
        s.ChannelMessageSendReply(m.ChannelID, "No active trivia game.", m.Reference())
        return
    }

    b.Trivia.End()
    s.ChannelMessageSend(m.ChannelID, "Trivia ended! Use `!!trivia scores` to see results.")
    log.Printf("Trivia ended by %s\n", m.Author.Username)
}

func (b *Bot) handleHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
    lines := []string{
        "**Trivia Bot Help**",
        "Here are the available commands:",
        "\n **User Commands:**",
        "- **!!trivia help**: Show this help message.",
        "- **!!trivia join <team>**: Join a team (e.g., `!!trivia join Red`).",
        "- **!!trivia answer <answer>**: Submit an answer to the current question (case-insensitive, e.g., `France` or `france`). Only the first correct answer earns points.",
        "- **!!trivia scores**: Display individual and team scores.",
        "\n**Admin Commands (restricted to the bot's admin user):**",
        "- **!!trivia start**: Start a new trivia contest.",
        "- **!!trivia end**: End the current trivia contest.",
        "- **!!trivia next**: Trigger the next question.",
        "- **!!trivia reset**: Reset all scores and teams, preserving questions.",
        "- **!!trivia list**: Post how many questiosn are in the database.",
        "- **!!trivia list questions**: Post all the questions in the database, without answers.",
        "- **!!trivia list answers**: Post all the questions in the database, with answers.",
        "- **!!trivia addq <question> | <answer>**: Add a new question (e.g., `!!trivia addq What is 2+2? | 4`).",
        "- **!!trivia removeq <id>**: Remove a question by ID.",
    }
    helpMessage := strings.Join(lines, "\n")

    s.ChannelMessageSendReply(m.ChannelID, helpMessage, m.Reference())
}

func (b *Bot) handleNext(s *discordgo.Session, m *discordgo.MessageCreate) {
    if !b.Trivia.Active {
        s.ChannelMessageSendReply(m.ChannelID, "No active trivia game. Use `!!trivia start` to begin.", m.Reference())
        return
    }

    // Signal the next question
    select {
    case b.Trivia.NextChan <- struct{}{}:
        // Success, question will be posted by runTrivia
    default:
        s.ChannelMessageSendReply(m.ChannelID, "A question is already being posted. Please wait.", m.Reference())
    }
    log.Printf("Next question requested by %s\n", m.Author.Username)
}

func (b *Bot) handleReset(s *discordgo.Session, m *discordgo.MessageCreate) {
    if err := b.DB.ResetScoresAndTeams(); err != nil {
        s.ChannelMessageSendReply(m.ChannelID, "Error resetting scores and teams.", m.Reference())
        log.Printf("Reset error: %v", err)
        return
    }

    // End any active trivia game
    if b.Trivia.Active {
        b.Trivia.End()
        s.ChannelMessageSend(m.ChannelID, "Trivia game ended.")
    }

    s.ChannelMessageSend(m.ChannelID, "Scores and teams reset successfully. Questions preserved.")
    log.Printf("Scores and teams reset by %s\n", m.Author.Username)
}

func (b *Bot) handleListQuestions(s *discordgo.Session, m *discordgo.MessageCreate, answerSwitch string) {
    includeAnswer := false

    switch answerSwitch {
    case "questions":
        includeAnswer = false
    case "answers":
        includeAnswer = true
    case "count":
        includeAnswer = false
    }
        
    questions, err := b.DB.ListQuestions()
    if err != nil {
        s.ChannelMessageSendReply(m.ChannelID, "Error fetching questions.", m.Reference())
        log.Printf("List questions error: %v", err)
        return
    }

    if len(questions) == 0 {
        s.ChannelMessageSendReply(m.ChannelID, "No questions in the database.", m.Reference())
        return
    }

    if answerSwitch == "count" {
        s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("There are %d questions in the database.", len(questions)), m.Reference())
        return
    }

    var response strings.Builder
    response.WriteString("**Question List**\n\n")
    for _, q := range questions {
        if !includeAnswer {
            q.Answer = "REDACTED"
        }
        line := fmt.Sprintf("ID: %d\nQuestion: %s\nAnswer: ||%s||\n\n", q.ID, q.Text, q.Answer)
        if response.Len()+len(line) > 1900 { // Reserve space for Discord's 2000-char limit
            s.ChannelMessageSend(m.ChannelID, response.String())
            response.Reset()
            response.WriteString("**Question List (continued)**\n\n")
        }
        response.WriteString(line)
    }

    if response.Len() > 0 {
        s.ChannelMessageSendReply(m.ChannelID, response.String(), m.Reference())
    }

    log.Printf("Questions listed by %s\n", m.Author.Username)
}
