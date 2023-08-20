package logger

import (
	"fmt"
	"log/syslog"
	"os"

	"github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
)

type Level int

const (
	LogDebug Level = iota
	LogInfo
	LogWarning
	LogError
	LogFatal
)

// Logger is a struct representing a custom logger.
type Logger struct {
	logger    *logrus.Logger
	debugMode bool
}

// NewLogger creates a new Logger object with logrus as the underlying logger.
// Returns a new Logger object.
func NewLogger() *Logger {
	logger := logrus.New()

	// Set up syslog hook
	hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, "")

	if err != nil {
		logger.WithFields(logrus.Fields{"prefix": "[ERROR]"}).Error("Unable to connect to local syslog daemon")
	} else {
		logger.AddHook(hook)
	}

	// Set default log level to Warning
	logger.SetLevel(logrus.InfoLevel)

	return &Logger{
		logger:    logger,
		debugMode: false,
	}
}

// Log writes a log message with the provided log level and fields.
// level: Level The log level of the message.
// message: string The log message.
// fields: map[string]interface{} The fields to be added to the log.
func (l *Logger) Log(level Level, message string, fields map[string]interface{}) {
	entry := l.logger.WithFields(fields)

	switch level {
	case LogDebug:
		fmt.Printf("DEBUG: %s\n", message)
	case LogInfo:
		entry.WithField("level", "[INFO]").Warn(message)
	case LogWarning:
		entry.WithField("level", "[WARN]").Warn(message)
	case LogError:
		entry.WithField("level", "[ERROR]").Error(message)
	case LogFatal:
		entry.WithField("level", "[FATAL]").Fatal(message)
	default:
		entry.Info(message)
	}

	if l.debugMode && level > LogInfo {
		fmt.Printf("DEBUG: %s\n", message)
	}
}

// SetDebugMode sets the debug mode of the logger.
// debugMode: bool The debug mode to set.
func (l *Logger) SetDebugMode(debugMode bool) {
	l.debugMode = debugMode
	if debugMode {
		l.logger.SetLevel(logrus.DebugLevel)
		l.logger.SetOutput(os.Stdout)
	} else {
		l.logger.SetLevel(logrus.InfoLevel)
		l.logger.SetOutput(os.Stdout)
	}
}
