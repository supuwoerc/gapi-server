package cronjob

import (
	"fmt"

	"github.com/supuwoerc/gapi-server/pkg/logger"

	"go.uber.org/zap"
)

type CronLogger struct {
	logger *logger.Logger
}

func NewCronLogger(l *logger.Logger) *CronLogger {
	return &CronLogger{logger: l}
}

func (c *CronLogger) Info(msg string, keysAndValues ...interface{}) {
	fields := make([]zap.Field, 0, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		fields = append(fields, zap.Any(fmt.Sprint(keysAndValues[i]), keysAndValues[i+1]))
	}
	c.logger.Info(fmt.Sprintf("cron: %s", msg), fields...)
}

func (c *CronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fields := []zap.Field{zap.Error(err)}
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		fields = append(fields, zap.Any(fmt.Sprint(keysAndValues[i]), keysAndValues[i+1]))
	}
	c.logger.Error(fmt.Sprintf("cron: %s", msg), fields...)
}
