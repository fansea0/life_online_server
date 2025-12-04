package ai

import (
	"context"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"life-online/pkg/eino"
)

func GetArkTextModel(modelCfg eino.ModelConfig) *ark.ChatModel {

	textModel, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		APIKey:      modelCfg.ModelApiKey,
		Region:      "cn-wuhan",
		Model:       modelCfg.ModelName,
		Temperature: volcengine.Float32(1.0),
		TopP:        volcengine.Float32(1.0),
	})
	if err != nil {
		logrus.WithError(err).Errorln("GetArkTextModel new ark model failed")
		return nil
	}
	return textModel
}
