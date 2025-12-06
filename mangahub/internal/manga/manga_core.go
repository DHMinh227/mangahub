package manga

import (
	"database/sql"
	"encoding/json"
	"errors"
	"mangahub/pkg/models"
)

func CoreGetManga(db *sql.DB, id string) (*models.Manga, error) {

	row := db.QueryRow(`
        SELECT id, title, author, genres, status, total_chapters, description
        FROM manga
        WHERE id = ?
    `, id)

	var (
		m          models.Manga
		genresJson string // temporary string, e.g. "[\"Drama\",\"Mystery\"]"
	)

	// Scan raw columns
	err := row.Scan(
		&m.ID,
		&m.Title,
		&m.Author,
		&genresJson, // <-- JSON string from DB
		&m.Status,
		&m.TotalChapters,
		&m.Description,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("manga not found")
	}
	if err != nil {
		return nil, err
	}

	// Convert JSON text → []string
	json.Unmarshal([]byte(genresJson), &m.Genres)

	return &m, nil
}
