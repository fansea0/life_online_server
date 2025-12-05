package game

import (
	"life-online/service/game"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	g := r.Group("/api/game")
	{
		g.POST("/action", GameAction)
	}
}

func GameAction(c *gin.Context) {
	var req game.UserAction
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求作为开始
		// if req.SessionID is empty, it will be handled by RunGameStep
	}

	resp, err := game.RunGameStep(req.SessionID, req.Choice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
