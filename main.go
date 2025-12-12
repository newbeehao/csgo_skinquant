package main

import (
	"flag"

	"newbeeHao.com/openapi/v2/conf"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// 读取命令行参数
	var env string
	flag.StringVar(&env, "e", "dev", "设置运行环境（如dev， prod）")
	flag.Parse()

	conf.InitConfig(env)

	// 启动服务器（默认 8080 端口）
	r.Run()
}
