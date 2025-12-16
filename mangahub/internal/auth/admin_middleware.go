package auth

import (
	"log"

	"github.com/gin-gonic/gin"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {

		claimsAny, ok := c.Get("claims")
		if !ok {
			c.JSON(401, gin.H{"error": "missing claims"})
			c.Abort()
			return
		}

		claims := claimsAny.(*Claims)
		log.Println("ADMIN CHECK ROLE =", claims.Role)

		if claims.Role != "admin" {
			c.JSON(403, gin.H{"error": "admin only"})
			c.Abort()
			return
		}

		c.Next()
	}
}
