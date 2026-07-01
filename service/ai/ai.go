package ai

import (
	"context"
	"time"

	"life-online/pkg/eino"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

const modelRequestTimeout = time.Minute

func GetArkTextModel(modelCfg eino.ModelConfig) *ark.ChatModel {
	textModel, err := ark.NewChatModel(context.Background(), buildArkChatModelConfig(modelCfg))
	if err != nil {
		logrus.WithError(err).Errorln("GetArkTextModel: new ark model failed")
		return nil
	}
	return textModel
}

func buildArkChatModelConfig(modelCfg eino.ModelConfig) *ark.ChatModelConfig {
	timeout := modelRequestTimeout
	retryTimes := 0
	return &ark.ChatModelConfig{
		APIKey:      modelCfg.ModelApiKey,
		Region:      "cn-wuhan",
		Model:       modelCfg.ModelName,
		Timeout:     &timeout,
		RetryTimes:  &retryTimes,
		Thinking:    &model.Thinking{Type: model.ThinkingTypeDisabled},
		Temperature: volcengine.Float32(1.0),
		TopP:        volcengine.Float32(1.0),
	}
}
