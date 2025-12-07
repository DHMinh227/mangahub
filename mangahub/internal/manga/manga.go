package manga

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Manga struct
type Manga struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	Genres        string `json:"genres"`
	Status        string `json:"status"`
	TotalChapters int    `json:"total_chapters"`
	Description   string `json:"description"`
}

// Latest update struct for endpoint
type LatestUpdate struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// RegisterRoutes = all manga routes in one file
func RegisterRoutes(r *gin.Engine, db *sql.DB) {

	// ---------------------------
	// GET /manga (all manga)
	// ---------------------------
	r.GET("/manga", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, title, author, genres, status, total_chapters, description FROM manga")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var list []Manga
		for rows.Next() {
			var m Manga
			rows.Scan(&m.ID, &m.Title, &m.Author, &m.Genres, &m.Status, &m.TotalChapters, &m.Description)
			list = append(list, m)
		}
		c.JSON(200, list)
	})

	// ---------------------------
	// GET /manga/:id
	// ---------------------------
	r.GET("/manga/:id", func(c *gin.Context) {
		id := c.Param("id")

		var m Manga
		err := db.QueryRow("SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE id = ?", id).
			Scan(&m.ID, &m.Title, &m.Author, &m.Genres, &m.Status, &m.TotalChapters, &m.Description)

		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		} else if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, m)
	})

	// ---------------------------
	// POST /manga (add manga)
	// ---------------------------
	r.POST("/manga", func(c *gin.Context) {
		var m Manga
		if err := c.BindJSON(&m); err != nil {
			c.JSON(400, gin.H{"error": "Invalid JSON"})
			return
		}

		_, err := db.Exec(`
			INSERT INTO manga (id, title, author, genres, status, total_chapters, description)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.ID, m.Title, m.Author, m.Genres, m.Status, m.TotalChapters, m.Description,
		)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Manga added successfully"})

		// AUTO UDP NOTIFICATION
		note := map[string]interface{}{
			"type":    "manga_added",
			"title":   m.Title,
			"message": "New manga added: " + m.Title,
		}

		data, _ := json.Marshal(note)
		http.Post("http://127.0.0.1:9094/broadcast", "application/json", bytes.NewBuffer(data))
	})

	// ---------------------------
	// PUT /manga/:id (update)
	// ---------------------------
	r.PUT("/manga/:id", func(c *gin.Context) {
		id := c.Param("id")

		var m Manga
		if err := c.BindJSON(&m); err != nil {
			c.JSON(400, gin.H{"error": "Invalid JSON"})
			return
		}

		_, err := db.Exec(`
			UPDATE manga
			SET title=?, author=?, genres=?, status=?, total_chapters=?, description=?
			WHERE id=?`,
			m.Title, m.Author, m.Genres, m.Status, m.TotalChapters, m.Description, id)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Manga updated successfully"})

		// AUTO UDP NOTIFICATION
		note := map[string]interface{}{
			"type":    "chapter_release",
			"title":   m.Title,
			"message": "New chapter released for " + m.Title,
		}

		data, _ := json.Marshal(note)
		http.Post("http://127.0.0.1:9094/broadcast", "application/json", bytes.NewBuffer(data))
	})

	// ---------------------------
	// DELETE /manga/:id
	// ---------------------------
	r.DELETE("/manga/:id", func(c *gin.Context) {
		id := c.Param("id")

		_, err := db.Exec("DELETE FROM manga WHERE id = ?", id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Manga deleted"})
	})

	// ---------------------------
	// GET /manga/latest-chapters
	// ---------------------------
	r.GET("/manga/latest-chapters", func(c *gin.Context) {

		rows, err := db.Query(`
			SELECT title, 'New chapter released for ' || title AS msg
			FROM manga
			ORDER BY id DESC
			LIMIT 10
		`)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var list []LatestUpdate

		for rows.Next() {
			var u LatestUpdate
			rows.Scan(&u.Title, &u.Message)
			list = append(list, u)
		}

		c.JSON(200, list)
	})
}
