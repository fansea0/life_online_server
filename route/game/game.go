package game

import (
	"github.com/sirupsen/logrus"
	"io"
	"life-online/service/game"
	"life-online/store/sanguo"
	"net/http"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func SetupRouter(r *gin.Engine) {
	g := r.Group("/api/game")
	{
		g.POST("/action", GameAction)
		g.POST("/choice", GameChoice)
		g.POST("/start", GameStart)
		g.GET("/ws", GameWS)
		g.GET("/identify_list", identifyList)
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func GameWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to upgrade connection")
		return
	}
	defer ws.Close()

	for {
		var req struct {
			Type      string `json:"type"` // "start" or "choice"
			Name      string `json:"name"`
			Identify  string `json:"identify"`
			SessionID string `json:"session_id"`
			Choice    string `json:"choice"`
		}

		err := ws.ReadJSON(&req)
		if err != nil {
			break
		}

		var streamReader *schema.StreamReader[*schema.Message]
		var sessionID string
		var currentContent string

		if req.Type == "start" {
			sessionID, streamReader, err = game.StartGameStream(req.Name, req.Identify)
			if err == nil {
				ws.WriteJSON(gin.H{"type": "session", "session_id": sessionID})
			}
		} else if req.Type == "choice" {
			sessionID = req.SessionID
			streamReader, err = game.HandleChoiceStream(req.SessionID, req.Choice)
		}

		if err != nil {
			ws.WriteJSON(gin.H{"type": "error", "message": err.Error()})
			continue
		}

		if streamReader != nil {
			// 读取流并推送
			for {
				chunk, err := streamReader.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}

				content := chunk.Content
				currentContent += content

				// 实时推送片段
				ws.WriteJSON(gin.H{
					"type":    "content",
					"content": content,
				})
			}
			streamReader.Close()
		}

		// 发送结束标记，告知前端本轮输出完毕
		ws.WriteJSON(gin.H{"type": "end"})

		// 更新完整上下文到内存
		game.UpdateContextWithResponse(sessionID, currentContent)
	}
}

func identifyList(c *gin.Context) {
	var req struct {
		Scope int `json:"scope" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	identities, err := sanguo.GetGameIdentitiesByScope(req.Scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"identities": identities})
}
