package user

import (
	"database/sql"
	"mangahub/internal/tcp"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterProgressRoutes(r gin.IRouter, db *sql.DB, emitter *tcp.ProgressEmitter) {

	r.POST("/users/progress", func(c *gin.Context) {

		userID := c.GetString("user_id")
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

		// 🔴 REAL-TIME PUSH (safe)
		if emitter != nil {
			_ = emitter.Emit(tcp.ProgressUpdate{
				UserID:    userID,
				MangaID:   req.MangaID,
				Chapter:   req.Chapter,
				Timestamp: time.Now().Unix(),
			})
		}

		c.JSON(200, gin.H{"message": "Progress saved"})
	})
}
