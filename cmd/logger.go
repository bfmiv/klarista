package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Logger - logger instance
var Logger *log.Logger

type prefixHook struct{}

func (l *prefixHook) Fire(entry *log.Entry) error {
	entry.Message = fmt.Sprintf("klarista: %s", entry.Message)
	return nil
}

func (l *prefixHook) Levels() []log.Level {
	return log.AllLevels
}

func init() {
	if isDebug() {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.AddHook(&prefixHook{})

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		CallerPrettyfier: func(frame *runtime.Frame) (string, string) {
			file := frame.File
			fileSegments := strings.Split(file, "/klarista/")
			if len(fileSegments) == 2 {
				file = fileSegments[1]
			}
			return fmt.Sprintf(" %s:%d", file, frame.Line), ""
		},
	})

	log.SetOutput(os.Stderr)

	Logger = log.StandardLogger()
}
