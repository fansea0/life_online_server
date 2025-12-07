package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

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

	// 打印本机 IP 提示
	printLocalIPs()

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

	// 根路径显示 index.html
	r.GET("/", func(c *gin.Context) {
		c.File("./resource/static/index.html")
	})

	// 使用 NoRoute 处理静态文件（当没有其他路由匹配时）
	r.NoRoute(func(c *gin.Context) {
		// 检查请求路径对应的文件是否存在
		requestPath := c.Request.URL.Path
		// 移除前导斜杠，防止路径遍历攻击
		if len(requestPath) > 0 && requestPath[0] == '/' {
			requestPath = requestPath[1:]
		}
		filePath := filepath.Join("./resource/static", filepath.Clean(requestPath))

		// 确保文件在 static 目录内（防止路径遍历）
		absStaticPath, _ := filepath.Abs("./resource/static")
		absFilePath, _ := filepath.Abs(filePath)
		if len(absFilePath) < len(absStaticPath) || absFilePath[:len(absStaticPath)] != absStaticPath {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}

		if _, err := os.Stat(filePath); err == nil {
			// 文件存在，直接返回
			c.File(filePath)
		} else {
			// 文件不存在，返回 404
			c.JSON(404, gin.H{"error": "Not found"})
		}
	})

	return r
}

func printLocalIPs() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	fmt.Println("------------------------------------------------")
	fmt.Println("Service is running at:")
	fmt.Println("- Local:   http://localhost:8080")
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Printf("- Network: http://%s:8080\n", ipnet.IP.String())
			}
		}
	}
	fmt.Println("------------------------------------------------")
}
