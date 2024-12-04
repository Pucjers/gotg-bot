package db

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type Voice struct {
	ID          int
	Name        string
	Description string
}

// The function `GetUserVoices` retrieves voices associated with a specific user from a database using
// a provided SQL query.
func GetUserVoices(db *sql.DB, userID int64) ([]Voice, error) {
	query := `SELECT id, name, description FROM voices WHERE author_id = $1`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve voices: %v", err)
	}
	defer rows.Close()

	var voices []Voice
	for rows.Next() {
		var voice Voice
		if err := rows.Scan(&voice.ID, &voice.Name, &voice.Description); err != nil {
			return nil, err
		}
		voices = append(voices, voice)
	}

	return voices, nil
}

// The function `SaveVoiceToDB` saves voice data to a database table.
func SaveVoiceToDB(db *sql.DB, voicePath, name, description string, tags []string, author string, authorID int64) error {
	query := `
		INSERT INTO voices (voice_path, name, description, tags, author, author_id)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := db.Exec(query, voicePath, name, description, pq.Array(tags), author, authorID)
	return err
}

func UpdateVoiceField(db *sql.DB, voiceID int, fieldName string, value interface{}) error {
	sqlStatement := fmt.Sprintf(`UPDATE voices SET %s = $1 WHERE id = $2`, fieldName)
	_, err := db.Exec(sqlStatement, value, voiceID)
	return err
}
