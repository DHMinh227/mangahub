package user

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterProgressRoutes(r *gin.Engine, db *sql.DB) {
	r.POST("/users/progress", func(c *gin.Context) {

		var req struct {
			UserID  string `json:"user_id"`
			MangaID string `json:"manga_id"`
			Chapter int    `json:"chapter"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		_, err := db.Exec(`
            INSERT INTO user_progress (user_id, manga_id, current_chapter)
            VALUES (?, ?, ?)
            ON CONFLICT(user_id, manga_id)
            DO UPDATE SET current_chapter = excluded.current_chapter
        `, req.UserID, req.MangaID, req.Chapter)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Progress updated"})
	})
}
