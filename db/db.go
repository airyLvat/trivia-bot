package db

import (
    "database/sql"
    "log"
    "os"
    "strings"

    _ "github.com/mattn/go-sqlite3"
)

type DB struct {
    *sql.DB
}

func NewDB() (*DB, error) {
    dbPath := os.Getenv("DATABASE_PATH")
    if dbPath == "" {
        dbPath = "./trivia.db" // Fallback for local development
    }
    log.Printf("Opening database at: %s", dbPath)
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    // Initialize tables
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS questions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            text TEXT,
            answer TEXT
        );
        CREATE TABLE IF NOT EXISTS players (
            user_id TEXT PRIMARY KEY,
            team TEXT,
            score INTEGER
        );
        CREATE TABLE IF NOT EXISTS teams (
            name TEXT PRIMARY KEY,
            score INTEGER
        );
    `)
    if err != nil {
        return nil, err
    }

    return &DB{db}, nil
}

func (db *DB) AddQuestion(text, answer string) error {
    text = strings.TrimSpace(text)
    answer = strings.TrimSpace(answer)
    _, err := db.Exec("INSERT INTO questions (text, answer) VALUES (?, ?)", text, answer)
    return err
}

func (db *DB) RemoveQuestion(id int) error {
    _, err := db.Exec("DELETE FROM questions WHERE id = ?", id)
    return err
}

func (db *DB) GetRandomQuestion() (*Question, error) {
    var q Question
    err := db.QueryRow("SELECT id, text, answer FROM questions ORDER BY RANDOM() LIMIT 1").Scan(&q.ID, &q.Text, &q.Answer)
    return &q, err
}

func (db *DB) JoinTeam(userID, team string) error {
    team = strings.ToLower(strings.TrimSpace(team))
    _, err := db.Exec("INSERT OR REPLACE INTO players (user_id, team, score) VALUES (?, ?, 0)", userID, team)
    if err != nil {
        return err
    }
    _, err = db.Exec("INSERT OR IGNORE INTO teams (name, score) VALUES (?, 0)", team)
    return err
}

func (db *DB) AddScore(userID, team string, points int) error {
    _, err := db.Exec("UPDATE players SET score = score + ? WHERE user_id = ?", points, userID)
    if err != nil {
        return err
    }
    _, err = db.Exec("UPDATE teams SET score = score + ? WHERE name = ?", points, team)
    return err
}

func (db *DB) GetScores() ([]Player, []Team, error) {
    players, err := db.Query("SELECT user_id, team, score FROM players ORDER BY score DESC")
    if err != nil {
        return nil, nil, err
    }
    defer players.Close()

    var playerList []Player
    for players.Next() {
        var p Player
        if err := players.Scan(&p.UserID, &p.Team, &p.Score); err != nil {
            return nil, nil, err
        }
        playerList = append(playerList, p)
    }

    teams, err := db.Query("SELECT name, score FROM teams ORDER BY score DESC")
    if err != nil {
        return nil, nil, err
    }
    defer teams.Close()

    var teamList []Team
    for teams.Next() {
        var t Team
        if err := teams.Scan(&t.Name, &t.Score); err != nil {
            return nil, nil, err
        }
        teamList = append(teamList, t)
    }

    return playerList, teamList, nil
}

func (db *DB) ResetScoresAndTeams() error {
    _, err := db.Exec("DELETE FROM teams")
    if err != nil {
        return err
    }
    _, err = db.Exec("DELETE FROM players")
    return err
}

func (db *DB) ListQuestions() ([]Question, error) {
    rows, err := db.Query("SELECT id, text, answer FROM questions ORDER BY id")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var questions []Question
    for rows.Next() {
        var q Question
        if err := rows.Scan(&q.ID, &q.Text, &q.Answer); err != nil {
            return nil, err
        }
        questions = append(questions, q)
    }

    return questions, nil
}
