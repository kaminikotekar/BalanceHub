package LBLog

import (
	"context"
	"github.com/kaminikotekar/BalanceHub/pkg/Config"
	"github.com/kaminikotekar/BalanceHub/pkg/Redis"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
)

const (
	logFileName = "/BalanceHub.log"
	WARNING     = "WARNING"
	INFO        = "INFO"
	ERROR       = "ERROR"
)

var (
	Initialized  = false
	messageQueue chan *Message
	WarningLog   *log.Logger
	InfoLog      *log.Logger
	ErrorLog     *log.Logger
	lbLogger     *LBLogger
)

type LBLogger struct {
	isRDLogger bool
	logFlag    map[string]*log.Logger
	logFile    string
}

func InitLogger() {
	isRDLogger := false
	if Redis.RedisEnabled {
		isRDLogger = true
	}

	file, err := os.OpenFile(Config.Configuration.LoadBalancer.AccessLogsPath+logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Error:  File Opening : ", err)
	}

	lbLogger = &LBLogger{
		isRDLogger: isRDLogger,
		logFlag: map[string]*log.Logger{
			INFO:    log.New(file, "INFO: ", log.Lshortfile),
			WARNING: log.New(file, "WARNING: ", log.Lshortfile),
			ERROR:   log.New(file, "ERROR: ", log.Lshortfile),
		},
		logFile: Config.Configuration.LoadBalancer.AccessLogsPath + logFileName,
	}
	messageQueue = getChannel()
	Initialized = true
}

func getChannel() chan *Message {
	if messageQueue == nil {
		messageQueue = make(chan *Message)
	}
	return messageQueue
}

func ProcessLogs() {
	if !Initialized {
		log.Println(WARNING, "Log worker not initialized !!")
		return
	}
	if lbLogger.isRDLogger {
		for {
			client := Redis.GetRDClient()
			ctx := Redis.GetContext()
			result, err := client.LPop(ctx, "logs").Result()
			if err == nil {
				// fmt.Println("Result: ", result)
				writeLog(unMarshal([]byte(result)))
				continue
			}
		}
	} else {
		for {
			msg, ok := <-messageQueue
			if !ok {
				log.Fatal("Could not read from channel : ", msg)
			}
			writeLog(msg)
		}
	}
}

func WriteRDMQ(ctx context.Context, client *redis.Client, msg []byte) {
	unMarshal(msg)
	if err := client.RPush(ctx, "logs", msg).Err(); err != nil {
		log.Println(ERROR, "Error while pushing: ", err)
	}
}

func writeLog(message *Message) {

	f, err := os.OpenFile(lbLogger.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	lbLogger.logFlag[message.Flag].Printf("%s %s", message.Timestamp, message.Message)
}
