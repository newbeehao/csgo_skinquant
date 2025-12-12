package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"newbeeHao.com/openapi/v2/common/constant"
)

type Logger struct {
	Entry *logrus.Entry
	Data  map[string]interface{}
}

func NewLogger(fileName ...string) *Logger {
	baseLogger := logrus.New()
	// 设置日志格式为自定义格式
	baseLogger.SetFormatter(&DIYFormatter{})
	// 默认开启调用栈记录
	baseLogger.SetReportCaller(true)
	baseLogger.AddHook(&CallerHook{Skip: 10})
	baseLogger.SetLevel(logrus.DebugLevel)
	entry := baseLogger.WithFields(logrus.Fields{})

	logger := &Logger{
		Entry: entry,
		Data:  make(map[string]interface{}),
	}

	// 设置日志级别，每个不同的日志级别会创建不同的LogFile对象
	// Trace Debug Print Info Warn Error Fatal Panic
	logger.Data[] = []string{"DEBUG", "INFO", "WARN", "ERROR"}
	if len(fileName) != 0 {
		logger.SetLoggerFileName(fileName[0])
	} else {
		logger.SetLoggerFileName("openapi")
	}

	return logger
}
func (log *Logger) SetLoggerFileName(prefix string) {
	log.Data[LOG_FILE_PREFIX_NAME] = prefix

	// // 根据实际支持的日志级别来添加
	// levels := log.GetValue(LEVELS).([]string)
	// for _, level := range levels {
	// 	logFilePath := log.GetLoggerPath(level)
	// 	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	// 	if err != nil {
	// 		logrus.Fatal(err)
	// 	}
	// 	log.LogFiles[level] = file
	// }
}

type DIYFormatter struct{}

func (f *DIYFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	// level := strings.ToUpper(entry.Level.String())
	message := entry.Message
	// 获取调用位置信息
	var file string
	var line int
	if entry.HasCaller() {
		file = entry.Caller.File
		line = entry.Caller.Line
		file = filepath.Base(file) // 只显示文件名
	}
	// 获取RequestID
	requestID := ""
	if id, ok := entry.Data[constant.RequestIDKey]; ok {
		requestID = fmt.Sprintf("%v", id)
	}

	// statusCode := ""
	// if id, ok := entry.Data[STATUS_CODE]; ok {
	// 	statusCode = fmt.Sprintf("%v", id)
	// }

	clientIP := ""
	if id, ok := entry.Data[CLIENT_IP]; ok {
		clientIP = fmt.Sprintf("%v", id)
	}

	method := ""
	if id, ok := entry.Data[METHOD]; ok {
		method = fmt.Sprintf("%v", id)
	}

	path := ""
	if id, ok := entry.Data[PATH]; ok {
		path = fmt.Sprintf("%v", id)
	}

	// 构建日志行
	logLine := fmt.Sprintf("%s [%s] [client_ip:%s, method:%s, path:%s] %s:%d",
		timestamp, requestID, clientIP, method, path, file, line)

	// if requestID != "" {
	// 	logLine += fmt.Sprintf(" [%s]", requestID)
	// }
	logLine += fmt.Sprintf(" : %s\n", message)

	return []byte(logLine), nil
}

// 定义调用栈Hook，调整skip
type CallerHook struct {
	Skip int
}

func (h *CallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *CallerHook) Fire(entry *logrus.Entry) error {
	// 获取正确的调用位置
	entry.Caller = getCaller(h.Skip)
	return nil
}

func getCaller(skip int) *runtime.Frame {
	pc := make([]uintptr, 1)
	n := runtime.Callers(skip, pc)
	if n < 1 {
		return nil
	}

	frame, _ := runtime.CallersFrames(pc).Next()
	// 简化文件路径
	if idx := strings.LastIndex(frame.File, "/"); idx != -1 {
		frame.File = frame.File[idx+1:]
	}
	return &frame
}

type LogRequest struct {
	FileName string // 日志文件名
	Data     []byte // 日志内容
}

// 日志存储的缓冲池
var LogChan = make(chan LogRequest, 10000)

// 自定义channel类型的io发送日志到channel
type ChannelWriter struct {
	FileName string // 日志文件名
}

func (w *ChannelWriter) Write(p []byte) (n int, err error) {
	// 发送到 Channel
	LogChan <- LogRequest{
		FileName: w.FileName,
		Data:     p,
	}
	return len(p), nil
}

// 消费日志池
func StartLogConsumer() {
	for {
		func() {
			files := make(map[string]*os.File)
			defer func() {
				// 程序退出时关闭所有文件
				for _, file := range files {
					file.Close()
				}
			}()

			for req := range LogChan {
				// 根据文件名获取文件
				file, ok := files[req.FileName]
				if !ok {
					var err error
					file, err = os.OpenFile(req.FileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
					if err != nil {
						fmt.Printf("Failed to open log file %s: %v\n", req.FileName, err)
						continue
					}
					files[req.FileName] = file
				}
				// 写入文件
				if _, err := file.Write(req.Data); err != nil {
					fmt.Printf("Failed to write log to %s: %v\n", req.FileName, err)
				}
			}
		}()

		// 消费协程挂了重启，避免日志缓冲池堆积，同时设置1s避免频繁重启
		time.Sleep(1 * time.Second)
	}
}

func (log *Logger) SetValue(key string, value interface{}) {
	log.Data[key] = value
}

func (log *Logger) GetValue(key string) interface{} {
	return log.Data[key]
}

func getRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(constant.RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

func getClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if clientIP, ok := ctx.Value(constant.CLIENT_IP).(string); ok {
		return clientIP
	}
	return ""
}

func getMethod(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if method, ok := ctx.Value(constant.METHOD).(string); ok {
		return method
	}
	return ""
}

func getPath(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if path, ok := ctx.Value(constant.PATH).(string); ok {
		return path
	}
	return ""
}
func (log *Logger) SetCommonFields(ctx context.Context) {
	log.Entry = log.Entry.WithFields(logrus.Fields{
		// STATUS_CODE: statusCode,
		CLIENT_IP:             getClientIP(ctx),
		METHOD:                getMethod(ctx),
		PATH:                  getPath(ctx),
		constant.RequestIDKey: getRequestID(ctx),
	})
}

// 根据日志级别获取当天对应的日志文件名（路径）
func (log *Logger) GetLoggerPath(level string) string {
	// cfg := config.Get()
	// // 确保日志目录存在
	// logDir := filepath.Dir(cfg.Log.File)
	err := os.MkdirAll(LogDir, 0755)
	if err != nil {
		fmt.Printf("Failed to create log directory: %v", err)
	}

	// logDir := "./log/"
	// logFilePath := fmt.Sprintf("%s%s.%s.%s.log", LogDir)

	fmt.Println("logDir:", LogDir)
	logFilePath := filepath.Join(LogDir, fmt.Sprintf("%s.%s.%s.log", log.Data[LOG_FILE_PREFIX_NAME].(string), level, time.Now().Format("2006-01-02")))
	fmt.Println("logFile:", logFilePath)
	return logFilePath
}

// 根据日志级别获取当天对应的日志文件
func (log *Logger) GetLogFile(level string) *os.File {
	// 查看当天的日志文件是否存在，若不存在，则创建
	logFilePath := log.GetLoggerPath(level)
	createLogFile(logFilePath)
	file, _ := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	return file
}

func createLogFile(filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		file, err := os.Create(filename)
		if err != nil {
			fmt.Printf("Failed to create log file %s: %v", filename, err)
		} else {
			fmt.Printf("Successfully created log file: %s", filename)
			file.Close()
		}
	} else {
		fmt.Printf("Log file already exists: %s", filename)
	}
}

func (log *Logger) Trace(ctx context.Context, args ...interface{}) {
	fileName := log.GetLoggerPath("TRACE")
	log.SetCommonFields(ctx)
	// log.Entry.Logger.SetOutput(os.Stdout)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Trace(args...)

}

func (log *Logger) Tracef(ctx context.Context, format string, args ...interface{}) {

	fileName := log.GetLoggerPath("TRACE")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Tracef(format, args...)
}

func (log *Logger) Debug(ctx context.Context, args ...interface{}) {

	fileName := log.GetLoggerPath("DEBUG")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Debug(args...)
}

func (log *Logger) Debugf(ctx context.Context, format string, args ...interface{}) {
	fileName := log.GetLoggerPath("DEBUG")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Debugf(format, args...)
}

func (log *Logger) Print(ctx context.Context, args ...interface{}) {

	fileName := log.GetLoggerPath("PRINT")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Print(args...)
}

func (log *Logger) Printf(ctx context.Context, format string, args ...interface{}) {

	fileName := log.GetLoggerPath("PRINT")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Printf(format, args...)
}

func (log *Logger) Info(ctx context.Context, args ...interface{}) {

	fileName := log.GetLoggerPath("INFO")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Info(args...)
}
func (log *Logger) Infof(ctx context.Context, format string, args ...interface{}) {

	fileName := log.GetLoggerPath("INFO")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Infof(format, args...)
}

func (log *Logger) Warn(ctx context.Context, args ...interface{}) {

	fileName := log.GetLoggerPath("WARN")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Warn(args...)
}

func (log *Logger) Warnf(ctx context.Context, format string, args ...interface{}) {

	fileName := log.GetLoggerPath("WARN")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Warnf(format, args...)
}

func (log *Logger) Error(ctx context.Context, args ...interface{}) {

	fileName := log.GetLoggerPath("ERROR")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Error(args...)
}

func (log *Logger) Errorf(ctx context.Context, format string, args ...interface{}) {

	fileName := log.GetLoggerPath("ERROR")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Errorf(format, args...)
}

func (log *Logger) Fatal(ctx context.Context, args ...interface{}) {

	fileName := log.GetLoggerPath("FATAL")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Fatal(args...)
}

func (log *Logger) Fatalf(ctx context.Context, format string, args ...interface{}) {

	fileName := log.GetLoggerPath("FATAL")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Fatalf(format, args...)
}

func (log *Logger) Panic(ctx context.Context, args ...interface{}) {

	fileName := log.GetLoggerPath("PANIC")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Panic(args...)
}

func (log *Logger) Panicf(ctx context.Context, format string, args ...interface{}) {

	fileName := log.GetLoggerPath("PANIC")
	log.SetCommonFields(ctx)
	log.Entry.Logger.SetOutput(&ChannelWriter{
		FileName: fileName,
	})
	log.Entry.Panicf(format, args...)
}
