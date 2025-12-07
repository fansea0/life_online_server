package game

import (
	"life-online/config"
	"life-online/pkg/eino"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
			"otherReqs":  config.GetOtherReqs(),
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

func StartGameStream(name, identify string) (string, *schema.StreamReader[*schema.Message], error) {
	msgContext, err := initSystemPrompt(name, identify)
	if err != nil {
		return "", nil, err
	}
	streamReader, err := eino.Stream(model, msgContext)
	if err != nil {
		logrus.WithError(err).Error("Stream failed")
		return "", nil, err
	}
	newUUID, _ := uuid.NewUUID()
	sessionID := newUUID.String()
	// 先保存当前的 Context (不含本次 AI 回复)
	SaveMsgContext(sessionID, msgContext)
	return sessionID, streamReader, nil
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

func HandleChoiceStream(sessionID, choice string) (*schema.StreamReader[*schema.Message], error) {
	context := GetMsgContext(sessionID)
	userMessage, err := eino.CreateMessagesCommon(choice, map[string]any{}, false)
	if err != nil {
		return nil, err
	}
	// 临时追加 userMessage 用于生成
	tempContext := append(context, userMessage...)

	streamReader, err := eino.Stream(model, tempContext)
	if err != nil {
		return nil, err
	}

	// 更新 Context 包含 UserMessage (此时尚未包含 AI 回复)
	SaveMsgContext(sessionID, tempContext)

	return streamReader, nil
}

// UpdateContextWithResponse 流结束后，将完整的 AI 响应追加到上下文
func UpdateContextWithResponse(sessionID string, fullResponse string) {
	context := GetMsgContext(sessionID)
	aiMsg := schema.AssistantMessage(fullResponse, nil)
	context = append(context, aiMsg)
	SaveMsgContext(sessionID, context)
}

func SaveMsgContext(sessionID string, msg []*schema.Message) {
	msgContextMap[sessionID] = msg
}

func GetMsgContext(sessionID string) []*schema.Message {
	return msgContextMap[sessionID]
}
