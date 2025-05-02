package db

type Question struct {
    ID       int
    Text     string
    Answer   string
}

type Player struct {
    UserID   string
    Team     string
    Score    int
}

type Team struct {
    Name     string
    Score    int
}
