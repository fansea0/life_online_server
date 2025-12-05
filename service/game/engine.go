package game

import (
	"encoding/json"
	"life-online/config"
	"life-online/pkg/eino"
	service_ai "life-online/service/ai"
	"regexp"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/sirupsen/logrus"
)

var (
	model *ark.ChatModel
)

func Init() {
	cfg := eino.ModelConfig{
		ModelName:   config.GetArkTextModelName(),
		ModelApiKey: config.GetArkTextModelApiKey(),
		ModelApiUrl: config.GetArkTextModelApiUrl(),
	}
	model = service_ai.GetArkTextModel(cfg)
}

// RunGameStep 执行一步游戏逻辑
func RunGameStep(sessionID string, choice string) (*GameResponse, error) {
	state := GetState(sessionID)
	if state == nil {
		// 如果没有 Session，新建一个
		state = CreateNewGame()
		choice = "开始新的人生" // 初始动作
	}

	// 1. 准备 Prompt 变量
	stateBytes, _ := json.Marshal(state)
	params := map[string]any{
		"StateJSON":  string(stateBytes),
		"UserChoice": choice,
	}

	// 2. 定义 Prompt
	promptCfg := &eino.PromptConfig{
		SystemMsg: `你是一个文字冒险游戏(Life Simulator)的Game Master。
你的任务是根据玩家当前状态和选择，生成下一段剧情，并给出接下来的选项。

请严格遵守以下规则：
1. **叙事风格**：使用第二人称("你")，代入感强，类似现代小说。根据年龄调整语气（婴儿期单纯，成年期复杂）。
2. **逻辑性**：根据玩家的属性(智力/体质/家境/快乐)和历史选择推演结果。
    - 属性低可能会导致失败或生病。
    - 属性高会解锁更好剧情。
    - 0岁时描述出生家庭环境。
3. **年龄推进**：每次行动后，根据剧情跨度适当增加年龄（0-10岁通常1-2岁一跳，成年后可更大跨度，如果发生重大事件可不增加）。
4. **输出格式**：
    - 第一部分：纯文本剧情描述（100-200字）。
    - 第二部分：在剧情结束后，必须换行输出一个 **JSON代码块** 用于程序解析。
    - JSON格式必须严格如下：
      ` + "```json" + `
      {
        "options": ["选项A描述", "选项B描述", "选项C描述"],
        "attr_change": {"智力": 1, "体质": -1, "快乐": 2, "age_add": 1}, 
        "game_over": false
      }
      ` + "```" + `
      (注意：age_add 是本次剧情增加的岁数，如果死亡或结局 game_over 为 true)`,
		UserMsg: `【玩家状态】：{{.StateJSON}}
【玩家选择】：{{.UserChoice}}

请生成接下来的剧情：`,
	}

	// 3. 生成消息链
	messages := eino.CreateMessages(promptCfg, params)

	// 4. 调用模型
	// 注意：这里使用 Generate 非流式，为了简单解析 JSON。生产环境可优化为 Stream 并分段解析。
	respMsg, err := eino.Generate(model, messages)
	if err != nil {
		return nil, err
	}

	content := respMsg.Content
	logrus.Infof("AI Response: %s", content)

	// 5. 解析结果
	story, aiResp := parseAIOutput(content)

	// 6. 更新状态
	updateState(state, aiResp)

	return &GameResponse{
		Story:      story,
		State:      state,
		Options:    aiResp.Options,
		AttrChange: aiResp.AttrChange,
	}, nil
}

func parseAIOutput(content string) (string, AIResponse) {
	// 简单解析：找到最后一个 ```json ... ``` 代码块
	var aiResp AIResponse
	story := content

	// 正则匹配 JSON 块
	re := regexp.MustCompile("(?s)```json(.*?)```")
	matches := re.FindAllStringSubmatch(content, -1)

	if len(matches) > 0 {
		jsonStr := matches[len(matches)-1][1] // 取最后一个匹配
		err := json.Unmarshal([]byte(jsonStr), &aiResp)
		if err != nil {
			logrus.WithError(err).Error("failed to unmarshal ai response json")
			// Fallback if JSON is bad
			aiResp.Options = []string{"继续"}
		}
		// 移除 JSON 块保留纯文本故事
		story = strings.Replace(content, matches[len(matches)-1][0], "", 1)
	} else {
		// 尝试找不带 markdown 的 json (如果模型没输出 code block)
		// 这里做一个简单的容错，如果没有 json，就默认继续
		logrus.Warn("no json block found in ai response")
		aiResp.Options = []string{"继续"}
	}

	return strings.TrimSpace(story), aiResp
}

func updateState(state *GameState, resp AIResponse) {
	// 更新属性
	for k, v := range resp.AttrChange {
		if k == "age_add" {
			state.Age += v
		} else {
			state.Attributes[k] += v
		}
	}
	state.IsGameOver = resp.GameOver
	SaveState(state)
}
