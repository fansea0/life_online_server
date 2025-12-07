package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

type envConfig struct {
	ArkTextModelName   string
	ArkTextModelApiKey string
	ArkTextModelApiUrl string

	SystemMsg  string
	RespFormat string
	OtherReqs  string
}

var (
	EnvConfigFile = "./config/env.toml"
	envConf       envConfig
)

func InitEnvConf() {
	if _, err := toml.DecodeFile(EnvConfigFile, &envConf); err != nil {
		log.Fatal(err)
	}
}

func GetArkTextModelName() string {
	return envConf.ArkTextModelName
}

func GetArkTextModelApiKey() string {
	return envConf.ArkTextModelApiKey
}

func GetArkTextModelApiUrl() string {
	return envConf.ArkTextModelApiUrl
}

func GetSystemMsg() string {
	return envConf.SystemMsg
}

func GetRespFormat() string {
	return envConf.RespFormat
}

func GetOtherReqs() string {
	return envConf.OtherReqs
}
