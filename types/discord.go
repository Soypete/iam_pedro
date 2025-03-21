package types

// DiscordAskMessage is metadata about a message that asks Pedro a question.
type DiscordAskMessage struct {
	ThreadID      string `db:"thread_id"`
	Message       string `db:"message"`
	Username      string `db:"username"`
	ThreadTimeout int    `db:"thread_timeout"`
	IsFromPedro   bool   `db:"is_from_pedro"`
}

// DO I WANT TO STORE THREAD HISTORY HERE?

// Discord20QuestionsMessage is metadata about a message that asks Pedro to play 20 questions.
type Discord20QuestionsMessage struct {
	GameID   string `db:"game_id"`
	ThreadID string `db:"thread_id"`
	Username string `db:"username"`
	Question string `db:"question"`
	Response string `db:"response"`
}

// Discord20QuestionsGame is metadata about a game of 20 questions.
type Discord20QuestionsGame struct {
	GameID        string `db:"game_id"`
	ThreadID      string `db:"thread_id"`
	Answer        string `db:"answer"`
	Username      string `db:"userid"`
	ThreadTimeout int    `db:"thread_timeout"`
	Status        string `db:"status"`
}
