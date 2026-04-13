package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/newbeehao/csgo_skinquant/internal/config"
)

func main() {
	// 1. 加载配置 (必填项缺失会直接 panic 退出)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	log.Printf("✅ 配置加载成功 (model=%s, port=%d)", cfg.DeepSeekModel, cfg.ServerPort)

	// 2. 初始化 Gin
	r := gin.Default()

	// 3. 健康检查端点 (用来确认服务真的活着)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "skinquant",
			"model":   cfg.DeepSeekModel,
		})
	})

	// 4. 启动 HTTP 服务
	addr := ":" + itoa(cfg.ServerPort)
	log.Printf("🚀 SkinQuant 启动, 监听 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("❌ 服务启动失败: %v", err)
	}
}

// itoa 小工具: int 转 string (避免引入 strconv 只为了一次转换)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
