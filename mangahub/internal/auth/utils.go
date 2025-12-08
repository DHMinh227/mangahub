package auth

import "github.com/gin-gonic/gin"

func GetUserID(c *gin.Context) (string, bool) {
	id, ok := c.Get("user_id")
	if !ok {
		return "", false
	}
	return id.(string), true
}
