package game

import (
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"life-online/config"
	"life-online/pkg/eino"
)

func init() {
	Init()
}

var (
	msgContextMap = make(map[string][]*schema.Message)
)

func initSystemPrompt(name, identify string) ([]*schema.Message, error) {

	msg, err := eino.CreateMessagesCommon(
		config.GetSystemMsg(),
		map[string]any{
			"name":       name,
			"identify":   identify,
			"respFormat": config.GetRespFormat(),
		},
		true,
	)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
func StartGame(name, identify string) (string, string, error) {
	msgContext, err := initSystemPrompt(name, identify)
	if err != nil {
		return "", "", err
	}
	rspMsg, err := eino.Generate(model, msgContext)
	if err != nil {
		logrus.WithError(err).Error("Generate failed")
		return "", "", err
	}
	newUUID, _ := uuid.NewUUID()
	// 加入msg到对话上下文
	msgContext = append(msgContext, rspMsg)
	// 加入缓存
	SaveMsgContext(newUUID.String(), msgContext)
	return newUUID.String(), rspMsg.Content, nil
}

func HandleChoice(sessionID, choice string) (string, error) {
	context := GetMsgContext(sessionID)
	userMessage, err := eino.CreateMessagesCommon(choice, map[string]any{}, false)
	if err != nil {
		return "", err
	}
	rspMsg, err := eino.Generate(model, append(context, userMessage...))
	if err != nil {
		return "", err
	}
	// 加入msg到对话上下文
	context = append(context, rspMsg)
	// 加入缓存
	SaveMsgContext(sessionID, context)
	return rspMsg.Content, nil
}

func SaveMsgContext(sessionID string, msg []*schema.Message) {
	msgContextMap[sessionID] = msg
}

func GetMsgContext(sessionID string) []*schema.Message {
	return msgContextMap[sessionID]
}
