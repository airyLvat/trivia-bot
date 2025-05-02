package bot

import (
    "sync"
    "time"
    "github.com/airylvat/trivia-bot/db"
)

type Trivia struct {
    Active     bool
    Current    *db.Question
    StartTime  time.Time
    NextChan   chan struct{}
    Mutex      sync.Mutex
}

func NewTrivia() *Trivia {
    return &Trivia{
        NextChan: make(chan struct{}),
    }
}

func (t *Trivia) Start() {
    t.Mutex.Lock()
    t.Active = true
    t.Mutex.Unlock()
}

func (t *Trivia) End() {
    t.Mutex.Lock()
    t.Active = false
    t.Current = nil
    t.Mutex.Unlock()
}

func (t *Trivia) SetQuestion(q *db.Question) {
    t.Mutex.Lock()
    t.Current = q
    t.StartTime = time.Now()
    t.Mutex.Unlock()
}
