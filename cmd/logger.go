package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/k0kubun/pp"
	log "github.com/sirupsen/logrus"
)

// Logger - logger instance
var Logger *log.Logger

// FormatStructOptions - format struct options
type FormatStructOptions struct {
	Compact bool
	Format  string
}

// FormatStruct - format struct as string
func FormatStruct(inputs ...interface{}) string {
	opts := FormatStructOptions{}
	values := []interface{}{}

	for _, input := range inputs {
		switch i := input.(type) {
		case FormatStructOptions:
			opts = i
		default:
			values = append(values, i)
		}
	}

	var results []string

	for _, value := range values {
		var result []byte
		var err error

		if opts.Format == "yaml" {
			result, err = yaml.Marshal(value)
		} else if opts.Format == "json" {
			if opts.Compact {
				result, err = json.Marshal(value)
			} else {
				result, err = json.MarshalIndent(value, "", "  ")
			}
		} else {
			result = []byte(pp.Sprint(value))
		}

		if err != nil {
			panic(err)
		}

		results = append(results, string(result))
	}

	return strings.Join(results, "\n")
}

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
