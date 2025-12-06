package manga

import (
	"database/sql"
	"encoding/json"
	"errors"
	"mangahub/pkg/models"
)

func CoreAddManga(db *sql.DB, m *models.Manga) error {
	// Check if manga already exists
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM manga WHERE id = ?", m.ID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists > 0 {
		return errors.New("manga already exists")
	}

	// Convert genres slice → JSON string
	genresJson, err := json.Marshal(m.Genres)
	if err != nil {
		return err
	}

	// Insert new manga
	_, err = db.Exec(`
        INSERT INTO manga (id, title, author, genres, status, total_chapters, description)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `,
		m.ID,
		m.Title,
		m.Author,
		string(genresJson),
		m.Status,
		m.TotalChapters,
		m.Description,
	)

	return err
}
