package game

import (
	"io"
	"life-online/service/game"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
		var summary string

		if req.Type == "start" {
			sessionID, streamReader, err = game.StartGameStream(req.Name, req.Identify)
			if err == nil {
				ws.WriteJSON(gin.H{"type": "session", "session_id": sessionID})
			}
		} else if req.Type == "choice" {
			sessionID = req.SessionID
			streamReader, err = game.HandleChoiceStream(req.SessionID, req.Choice)
		}
		logrus.WithField("choice", req.Choice).Infoln("game choice")

		if err != nil {
			ws.WriteJSON(gin.H{"type": "error", "message": err.Error()})
			continue
		}

		if streamReader != nil {
			// 读取流并推送
			summaryStarted := false
			for {
				chunk, err := streamReader.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}

				content := chunk.Content

				// 检查是否开始总结内容（以&开头）
				if strings.Contains(content, "&") && !summaryStarted {
					// 找到@的位置
					atIndex := strings.Index(content, "&")
					summary = content[atIndex+1:] // 不包含&
					if atIndex >= 0 {
						// 发送@之前的部分给前端
						contentToSend := content[:atIndex] // 不包含@
						if contentToSend != "" {
							ws.WriteJSON(gin.H{
								"type":    "content",
								"content": contentToSend,
							})
						}
						summaryStarted = true
						// @之后的内容不发送给前端
						continue
					}
				}

				// 如果还没有开始总结，正常发送内容
				ws.WriteJSON(gin.H{
					"type":    "content",
					"content": content,
				})

				if summaryStarted {
					summary += content
				}
			}
			streamReader.Close()
		}

		// 发送结束标记，告知前端本轮输出完毕
		ws.WriteJSON(gin.H{"type": "end"})
		logrus.WithFields(logrus.Fields{
			"summary": summary,
		}).Infoln("game summary")

		// 更新完整上下文到内存
		game.UpdateContextWithResponse(sessionID, summary)
	}
}

func identifyList(c *gin.Context) {
	// 暂时只支持scope=1的三国身份列表
	_ = c.Query("scope") // 保留查询参数以备将来扩展

	// 硬编码的三国身份选项
	identities := []gin.H{
		{"id": 1, "description": "魏国的一名普通士兵", "uid": 0, "scope": 1},
		{"id": 2, "description": "魏国的一名年轻将领", "uid": 0, "scope": 1},
		{"id": 3, "description": "魏国的一名谋士", "uid": 0, "scope": 1},
		{"id": 4, "description": "蜀国的一名普通士兵", "uid": 0, "scope": 1},
		{"id": 5, "description": "蜀国的一名年轻将领", "uid": 0, "scope": 1},
		{"id": 6, "description": "蜀国的一名谋士", "uid": 0, "scope": 1},
		{"id": 7, "description": "吴国的一名普通士兵", "uid": 0, "scope": 1},
		{"id": 8, "description": "吴国的一名年轻将领", "uid": 0, "scope": 1},
		{"id": 9, "description": "吴国的一名谋士", "uid": 0, "scope": 1},
		{"id": 10, "description": "黄巾起义军的一名小将", "uid": 0, "scope": 1},
		{"id": 11, "description": "袁绍军的一名将领", "uid": 0, "scope": 1},
		{"id": 12, "description": "刘表荆州军的一名官员", "uid": 0, "scope": 1},
		{"id": 13, "description": "马腾西凉军的一名骑兵", "uid": 0, "scope": 1},
		{"id": 14, "description": "公孙瓒白马义从的一员", "uid": 0, "scope": 1},
		{"id": 15, "description": "张鲁五斗米教的信徒", "uid": 0, "scope": 1},
		{"id": 16, "description": "韩遂西凉叛军的首领", "uid": 0, "scope": 1},
		{"id": 17, "description": "陶谦徐州军的郡守", "uid": 0, "scope": 1},
		{"id": 18, "description": "刘璋益州军的官员", "uid": 0, "scope": 1},
	}

	c.JSON(http.StatusOK, gin.H{"identities": identities})
}
