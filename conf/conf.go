package conf

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	config        Config
	configMutex   sync.RWMutex // 用于保护config变量的读写锁
	lastEventTime time.Time    // 记录最后一次配置变更事件的时间，用于防抖
)

type Config struct {
	App   App   `mapstructure:"app"`
	Mysql Mysql `mapstructure:"mysql"`
}

type App struct {
	Port        string `mapstructure:"port"`
	ReadTimeout int    `mapstructure:"read_timeout"`
}

type Mysql struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	DatabaseName string `mapstructure:"database_name"`
}

type Log struct {
	LogPath string `mapstructure:"log_path"`
}

// 目前支持 dev/prod
func InitConfig(env string) error {
	v := viper.New()
	v.SetConfigName("config." + env)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("无法读取配置文件: %v", err)
		return fmt.Errorf("无法读取配置文件: %v", err)
	}
	log.Printf("已加载配置文件: %s", v.ConfigFileUsed())

	if err := v.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode config into struct: %v", err)
	}

	// 3. 监听配置文件变更
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		// 防抖处理：1秒内多次变更只处理一次
		if time.Since(lastEventTime) < time.Second {
			return
		}
		lastEventTime = time.Now()

		log.Println("检测到配置文件变更，重新加载:", e.Name)

		// 安全地重新加载配置
		if err := safeReloadConfig(v); err != nil {
			log.Printf("热加载配置出错: %v", err)
			// 此处可以添加警报逻辑，但不要让程序崩溃
		} else {
			log.Println("配置热更新成功！")
		}
	})
	return nil
}

// 提供一个安全的函数来获取配置
func GetConfig() Config {
	configMutex.RLock()         // 获取读锁
	defer configMutex.RUnlock() // 确保函数返回前释放锁
	return config               // 返回配置的副本
}

func safeReloadConfig(v *viper.Viper) error {
	// 创建一个临时变量来接收新配置
	var newConfig Config

	// Viper 会读取更新后的文件内容到内存
	if err := v.Unmarshal(&newConfig); err != nil {
		return err
	}

	// 如果需要，可以在这里添加对新配置的验证逻辑
	// 例如：if newConfig.App.Port == "" { return errors.New("端口号不能为空") }

	// 获取写锁，原子性地更新全局配置
	configMutex.Lock()
	defer configMutex.Unlock()
	config = newConfig

	return nil
}
