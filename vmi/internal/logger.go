package vmi_internal

import (
	"github.com/bgp59/logrusx"
	"github.com/sirupsen/logrus"
)

var RootLogger = logrusx.NewCollectableLogger()

// Public access to the root logger, needed for testing:
func GetRootLogger() *logrusx.CollectableLogger { return RootLogger }

func init() {
	// Add the default prefix for the current module, which is 2 dirs up from
	// here.
	RootLogger.AddCallerSrcPathPrefix(2)
}

// Set the logger based on config:
func SetLogger(logCfg *logrusx.LoggerConfig) error {
	return RootLogger.SetLogger(logCfg)
}

func NewCompLogger(compName string) *logrus.Entry {
	return RootLogger.NewCompLogger(compName)
}
