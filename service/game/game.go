package game

import (
	"life-online/config"
	"life-online/pkg/eino"

	"github.com/cloudwego/eino/schema"
	"github.com/sirupsen/logrus"
)

func init() {
	Init()
}

var (
	msgContextMap = make(map[string][]*schema.Message)
)

const initialGamePrompt = "开始游戏，请根据我的姓名和初始身份生成开局剧情，并在结尾提供三个可选行动。"

func initSystemPrompt(name, identify, affinity, stateHint, turnKindRule string) ([]*schema.Message, error) {
	msg, err := eino.CreateMessagesCommon(
		config.GetSystemMsg(),
		map[string]any{
			"name":             name,
			"identify":         identify,
			"identityAffinity": affinity,
			"stateHint":        stateHint,
			"turnKindRule":     turnKindRule,
			"respFormat":       config.GetRespFormat(),
			"otherReqs":        config.GetOtherReqs(),
		},
		true,
	)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// buildTurnKindRule 根据 state + RecentKinds 算出候选回合类型并拼接规则段
func buildTurnKindRule(state *GameState) string {
	if state == nil {
		return kindPromptRule("normal")
	}
	candidate := decideKindCandidate(state, state.RecentKinds)
	return "本轮建议回合类型：" + candidate + "。" + kindPromptRule(candidate) + "你必须按此类型生成剧情与选项，并在 kind 字段回填实际类型。"
}

func appendInitialGamePrompt(msgContext []*schema.Message) []*schema.Message {
	return append(msgContext, schema.UserMessage(initialGamePrompt))
}

func StartGame(name, identify string) (string, string, error) {
	state := CreateNewGame(name, identify)
	affinity := state.IdentityAffinity
	msgContext, err := initSystemPrompt(name, identify, affinity, "", kindPromptRule("normal"))
	if err != nil {
		return "", "", err
	}
	msgContext = appendInitialGamePrompt(msgContext)
	rspMsg, err := eino.Generate(model, msgContext)
	if err != nil {
		logrus.WithError(err).Error("StartGame: eino.Generate failed")
		return "", "", err
	}
	// 用 state 的 SessionID 作为对外 sessionID，保证状态与上下文键一致
	sessionID := state.SessionID
	// 加入msg到对话上下文
	msgContext = append(msgContext, rspMsg)
	// 加入缓存
	SaveMsgContext(sessionID, msgContext)
	return sessionID, rspMsg.Content, nil
}

func StartGameStream(name, identify string) (string, *schema.StreamReader[*schema.Message], error) {
	state := CreateNewGame(name, identify)
	affinity := state.IdentityAffinity
	// 开局首轮 RecentKinds 为空，候选强制 normal
	msgContext, err := initSystemPrompt(name, identify, affinity, "", kindPromptRule("normal"))
	if err != nil {
		return "", nil, err
	}
	msgContext = appendInitialGamePrompt(msgContext)
	streamReader, err := eino.Stream(model, msgContext)
	if err != nil {
		logrus.WithError(err).Error("StartGameStream: eino.Stream failed")
		return "", nil, err
	}
	// 用 state 的 SessionID 作为对外 sessionID，保证状态与上下文键一致
	sessionID := state.SessionID
	// 保存上下文（不含本次 AI 回复）
	SaveMsgContext(sessionID, msgContext)
	return sessionID, streamReader, nil
}

func HandleChoice(sessionID, choice string) (string, error) {
	context := GetMsgContext(sessionID)
	state := GetState(sessionID)
	stateHint := ""
	affinity := ""
	turnKindRule := kindPromptRule("normal")
	if state != nil {
		stateHint = stateHintFromState(state)
		affinity = state.IdentityAffinity
		turnKindRule = buildTurnKindRule(state)
	}
	// 重新生成带最新状态提示的 system，替换上下文首条 system 消息
	sysMsg, err := initSystemPrompt(stateIdentityName(state), stateIdentityDesc(state), affinity, stateHint, turnKindRule)
	if err == nil && len(sysMsg) > 0 && len(context) > 0 {
		// 替换首条 system，保留后续历史
		context = append([]*schema.Message{sysMsg[0]}, context[1:]...)
	}
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
	state := GetState(sessionID)
	stateHint := ""
	affinity := ""
	turnKindRule := kindPromptRule("normal")
	if state != nil {
		stateHint = stateHintFromState(state)
		affinity = state.IdentityAffinity
		turnKindRule = buildTurnKindRule(state)
	}
	// 重新生成带最新状态提示的 system，替换上下文首条 system 消息
	sysMsg, err := initSystemPrompt(stateIdentityName(state), stateIdentityDesc(state), affinity, stateHint, turnKindRule)
	if err == nil && len(sysMsg) > 0 && len(context) > 0 {
		// 替换首条 system，保留后续历史
		context = append([]*schema.Message{sysMsg[0]}, context[1:]...)
	}
	userMessage, err := eino.CreateMessagesCommon(choice, map[string]any{}, false)
	if err != nil {
		return nil, err
	}
	tempContext := append(context, userMessage...)
	streamReader, err := eino.Stream(model, tempContext)
	if err != nil {
		return nil, err
	}
	SaveMsgContext(sessionID, tempContext)
	return streamReader, nil
}

// stateIdentityName 从 state 取姓名，state 缺失时兜底
func stateIdentityName(state *GameState) string {
	if state != nil {
		return state.Name
	}
	return "你"
}

// stateIdentityDesc 从 state 取身份描述
func stateIdentityDesc(state *GameState) string {
	if state != nil {
		return state.Identity
	}
	return ""
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
