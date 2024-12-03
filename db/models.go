package db

type FSMVoices struct {
	Voice       string
	Name        string
	Description string
	Tags        []string
	Author      string
	AuthorID    int64
	State       string
}
