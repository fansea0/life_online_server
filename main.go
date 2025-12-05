package main

import (
	"github.com/gin-gonic/gin"
	"life-online/config"
	"life-online/route/ai"
	"life-online/route/game"
	service_game "life-online/service/game"
)

func init() {
	config.InitEnvConf()
	service_game.Init()
}
func main() {

	// 1. 初始化配置

	// 1.1 数据库配置

	// 2. 注册路由
	engine := setupRouter()
	// 3. 启动服务
	err := engine.Run(":8080")
	if err != nil {
		panic(err)
		return
	}

}

func setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	ai.SetupRouter(r)
	game.SetupRouter(r)
	r.StaticFile("/", "./resource/static/index.html")
	return r
}
