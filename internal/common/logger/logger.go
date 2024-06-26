package logger

import (
	"runtime"

	"github.com/evalphobia/logrus_sentry"
	"github.com/sirupsen/logrus"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/config"
)

var _Logger *logrus.Logger
var _CustomLogger *logrus.Entry

func SetGlobalLogger(conf *config.Config) {
	var _logger *logrus.Logger
	var _level logrus.Level
	var err error

	_logger = logrus.New()
	// Set log level.
	if len(conf.LogLevel) > 0 {
		_level, err = logrus.ParseLevel(conf.LogLevel)
		if err != nil {
			_level = logrus.DebugLevel
		}
	} else {
		_level = logrus.DebugLevel
	}
	_logger.SetLevel(_level)
	// Set log formatter.
	_logger.SetFormatter(&logrus.TextFormatter{})
	// Set hook for sentry.
	if len(conf.LogSentryDSN) > 0 {
		hook, err := logrus_sentry.NewSentryHook(conf.LogSentryDSN, []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		})
		if err != nil {
			logrus.WithError(err).Fatal("Failed to set sentry hook for logrus logger.")
		} else {
			_logger.Hooks.Add(hook)
		}
	}
	_Logger = _logger
	// Set log fields.
	_CustomLogger = _logger.WithField("service_name", conf.ServiceName)
	_CustomLogger.Debugf(">>> runtime.NumCPU:%d", runtime.NumCPU())
	_CustomLogger.Debugf(">>> runtime.GOMAXPROCS:%d", runtime.GOMAXPROCS(8))
}

func GetGlobalLogger() *logrus.Entry {
	return _CustomLogger
}

func GetStandardLogger() *logrus.Logger {
	return _Logger
}
