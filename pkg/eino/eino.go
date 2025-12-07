package eino

import (
	"context"
	"errors"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/sirupsen/logrus"
	"io"
	"log"
)

type ModelConfig struct {
	ModelName   string `json:"name"`
	ModelApiKey string `json:"api_key"`
	ModelApiUrl string `json:"api_url"`
}

type PromptConfig struct {
	SystemMsg string `json:"system_msg"`
	UserMsg   string `json:"user_msg"`
	//AdditionMsg string `json:"addition_msg"` // 追问消息
}

func CreateMessages(promptCfg *PromptConfig, params map[string]any) []*schema.Message {
	if promptCfg == nil {
		return nil
	}

	// 选择 GoTemplate 模板引擎,优点在于可以避免prompt包含json语句导致模板失效
	// GoTemplate 接收模板语法为{{.field}},与Go语言语法一致
	messages, err := prompt.FromMessages(schema.GoTemplate,
		// 系统消息模板
		schema.SystemMessage(promptCfg.SystemMsg),
		// 用户消息模板
		schema.UserMessage(promptCfg.UserMsg),
	).Format(context.Background(), params)

	if err != nil {
		logrus.WithError(err).Errorln("format template failed")
	}
	return messages
}
func CreateMessagesCommon(prompts string, params map[string]any, isSystem bool) ([]*schema.Message, error) {
	message := schema.SystemMessage(prompts)
	if !isSystem {
		message = schema.UserMessage(prompts)
	}
	// 选择 GoTemplate 模板引擎,优点在于可以避免prompt包含json语句导致模板失效
	// GoTemplate 接收模板语法为{{.field}},与Go语言语法一致
	messages, err := prompt.FromMessages(schema.GoTemplate,
		message,
	).Format(context.Background(), params)

	if err != nil {
		logrus.WithError(err).Errorln("format template failed")
		return nil, err
	}
	return messages, nil
}

func Generate(model *ark.ChatModel, messages []*schema.Message, option ...model.Option) (*schema.Message, error) {
	if model == nil {
		return nil, errors.New("model is nil")
	}
	if len(messages) == 0 {
		return nil, errors.New("messages is empty")
	}
	return model.Generate(context.Background(), messages, option...)
}

func Stream(model *ark.ChatModel, messages []*schema.Message, option ...model.Option) (outputStream *schema.StreamReader[*schema.Message], err error) {
	if model == nil {
		return nil, errors.New("model is nil")
	}
	if len(messages) == 0 {
		return nil, errors.New("messages is empty")
	}
	return model.Stream(context.Background(), messages, option...)
}

func ReportStream(sr *schema.StreamReader[*schema.Message]) {
	defer sr.Close()

	i := 0
	for {
		message, err := sr.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("recv failed: %v", err)
		}
		log.Printf("message[%d]: %+v\n", i, message)
		i++
	}
}
