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
		g.POST("/choice", GameChoice)
		//g.POST("/choice_stream", GameChoiceStream)
		g.POST("/start", GameStart)
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

func GameChoice(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id"`
		Choice    string `json:"choice"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	content, err := game.HandleChoice(req.SessionID, req.Choice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"content": content,
	})
}

func GameStart(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		Identify string `json:"identify"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, "参数错误")
		return
	}
	uuid, content, err := game.StartGame(req.Name, req.Identify)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"session_id": uuid, "content": content})
}
