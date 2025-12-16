package manga

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mangahub/internal/udp"
)

func RegisterAdminRoutes(
	r *gin.RouterGroup,
	db *sql.DB,
	udpServer *udp.NotificationServer,
) {

	r.POST("/manga", func(c *gin.Context) {
		var m Manga
		if err := c.BindJSON(&m); err != nil {
			c.JSON(400, gin.H{"error": "invalid json"})
			return
		}

		genres := strings.Join(m.Genres, ",")

		_, err := db.Exec(`
			INSERT INTO manga (id, title, author, genres, status, total_chapters, description)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.ID,
			m.Title,
			m.Author,
			genres,
			m.Status,
			m.TotalChapters,
			m.Description,
		)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		fmt.Println("📢 Broadcast called")
		fmt.Printf("UDP SERVER POINTER: %+v\n", udpServer)

		// ✅ DIRECT UDP NOTIFY
		udpServer.Broadcast(udp.Notification{
			Type:      "NEW_MANGA",
			MangaID:   m.ID,
			Message:   "New manga added: " + m.Title,
			Timestamp: time.Now().Unix(),
		})

		c.JSON(201, gin.H{"message": "manga added"})
	})

	r.DELETE("/manga/:id", func(c *gin.Context) {
		id := c.Param("id")

		if _, err := db.Exec(`DELETE FROM manga WHERE id = ?`, id); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "manga deleted"})
	})
}
