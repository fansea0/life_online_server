package main

import (
	"github.com/gin-gonic/gin"
	"life-online/route/ai"
)

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
	return r
}
