package manga

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Manga struct matches your database table
type Manga struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	Genres        string `json:"genres"`
	Status        string `json:"status"`
	TotalChapters int    `json:"total_chapters"`
	Description   string `json:"description"`
}

// RegisterRoutes sets up /manga routes
func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	r.GET("/manga", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, title, author, genres, status, total_chapters, description FROM manga")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var mangas []Manga
		for rows.Next() {
			var m Manga
			rows.Scan(&m.ID, &m.Title, &m.Author, &m.Genres, &m.Status, &m.TotalChapters, &m.Description)
			mangas = append(mangas, m)
		}
		c.JSON(http.StatusOK, mangas)
	})

	r.GET("/manga/:id", func(c *gin.Context) {
		id := c.Param("id")
		var m Manga
		err := db.QueryRow("SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE id = ?", id).
			Scan(&m.ID, &m.Title, &m.Author, &m.Genres, &m.Status, &m.TotalChapters, &m.Description)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Manga not found"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, m)
	})

	r.POST("/manga", func(c *gin.Context) {
		var m Manga
		if err := c.BindJSON(&m); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		_, err := db.Exec(`INSERT INTO manga (id, title, author, genres, status, total_chapters, description)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, m.ID, m.Title, m.Author, m.Genres, m.Status, m.TotalChapters, m.Description)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Manga added successfully"})
	})

	r.PUT("/manga/:id", func(c *gin.Context) {
		id := c.Param("id")
		var m Manga
		if err := c.BindJSON(&m); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		_, err := db.Exec(`UPDATE manga SET title=?, author=?, genres=?, status=?, total_chapters=?, description=? WHERE id=?`,
			m.Title, m.Author, m.Genres, m.Status, m.TotalChapters, m.Description, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Manga updated successfully"})
	})

	r.DELETE("/manga/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.Exec("DELETE FROM manga WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Manga deleted"})
	})
}
