package ai

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	api := r.Group("ai/generate/")
	{
		api.GET("random_event", RandomEvent)
	}
}

func RandomEvent(c *gin.Context) {
	var req struct {
		EventId int `form:"event_id"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "ok"})
}
