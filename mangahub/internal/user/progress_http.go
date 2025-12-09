package user

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterProgressRoutes(r gin.IRouter, db *sql.DB) {

	r.POST("/users/progress", func(c *gin.Context) {

		// user ID extracted by JWT middleware
		userID := c.GetString("userID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req struct {
			MangaID string `json:"manga_id"`
			Chapter int    `json:"chapter"`
		}

		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid JSON"})
			return
		}

		if req.MangaID == "" || req.Chapter <= 0 {
			c.JSON(400, gin.H{"error": "Missing manga_id or chapter"})
			return
		}

		_, err := db.Exec(`
            INSERT INTO user_progress(user_id, manga_id, current_chapter)
            VALUES(?, ?, ?)
            ON CONFLICT(user_id, manga_id)
            DO UPDATE SET current_chapter = excluded.current_chapter
        `, userID, req.MangaID, req.Chapter)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Progress saved"})
	})
}
