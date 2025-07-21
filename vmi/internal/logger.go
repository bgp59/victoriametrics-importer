package vmi_internal

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LOGGER_CONFIG_USE_JSON_DEFAULT                = true
	LOGGER_CONFIG_LEVEL_DEFAULT                   = "info"
	LOGGER_CONFIG_DISBALE_SRC_FILE_DEFAULT        = false
	LOGGER_CONFIG_LOG_FILE_DEFAULT                = "" // i.e. stderr
	LOGGER_CONFIG_LOG_FILE_MAX_SIZE_MB_DEFAULT    = 10
	LOGGER_CONFIG_LOG_FILE_MAX_BACKUP_NUM_DEFAULT = 1

	LOGGER_DEFAULT_LEVEL    = logrus.InfoLevel
	LOGGER_TIMESTAMP_FORMAT = time.RFC3339
	// Extra field added for component sub loggers:
	LOGGER_COMPONENT_FIELD_NAME = "comp"
)

// Collectable logger interface for logurs.Log (see vmitestutil/log_collector.go):
type CollectableLogger struct {
	logrus.Logger
	// Cache the condition of being enabled for debug or not. Various sections
	// of  the code may test this condition before doing more expensive actions,
	// such as formatting debug info, so it pays off to make it as efficient as
	// possible:
	IsEnabledForDebug bool
}

func (log *CollectableLogger) GetOutput() io.Writer {
	return log.Out
}

func (log *CollectableLogger) GetLevel() any {
	return log.Logger.GetLevel()
}

func (log *CollectableLogger) SetLevel(level any) {
	if level, ok := level.(logrus.Level); ok {
		log.Logger.SetLevel(level)
		log.IsEnabledForDebug = log.IsLevelEnabled(logrus.DebugLevel)
	}
}

type LoggerConfig struct {
	// Whether to structure the logged record in JSON:
	UseJson bool `yaml:"use_json"`
	// Log level name: info, warn, ...:
	Level string `yaml:"level"`
	// Whether to disable the reporting of the source file:line# info:
	DisableSrcFile bool `yaml:"disable_src_file"`
	// Whether to log to a file or, if empty, to stderr:
	LogFile string `yaml:"log_file"`
	// Log file max size, in MB, before rotation, use 0 to disable:
	LogFileMaxSizeMB int `yaml:"log_file_max_size_mb"`
	// How many older log files to keep upon rotation:
	LogFileMaxBackupNum int `yaml:"log_file_max_backup_num"`
}

func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		UseJson:             LOGGER_CONFIG_USE_JSON_DEFAULT,
		Level:               LOGGER_CONFIG_LEVEL_DEFAULT,
		DisableSrcFile:      LOGGER_CONFIG_DISBALE_SRC_FILE_DEFAULT,
		LogFile:             LOGGER_CONFIG_LOG_FILE_DEFAULT,
		LogFileMaxSizeMB:    LOGGER_CONFIG_LOG_FILE_MAX_SIZE_MB_DEFAULT,
		LogFileMaxBackupNum: LOGGER_CONFIG_LOG_FILE_MAX_BACKUP_NUM_DEFAULT,
	}
}

// When files are logged, the file name is converted to a relative path,
// generally based on the root dir of the module. This package is used within
// this module and it also imported from other modules. Each importer should
// declare its dir path prefix and the longest match will be used.

type ModuleDirPathCache struct {
	// List of prefixes to be removed from the file path when logging, sorted in
	// reverse order by length.
	prefixList []string
	// If no prefix match is found, the number of directories to keep from the
	// end of the path.
	keepNDirs int
}

func (p *ModuleDirPathCache) addPrefix(prefix string) error {
	i := len(p.prefixList) - 1
	for i >= 0 {
		if p.prefixList[i] == prefix {
			return nil // already there
		}
		if len(p.prefixList[i]) > len(prefix) {
			break
		}
		i--
	}
	i++
	if i >= len(p.prefixList) {
		p.prefixList = append(p.prefixList, prefix)
	} else {
		p.prefixList = append(p.prefixList[:i+1], p.prefixList[i:]...)
		p.prefixList[i] = prefix
	}
	return nil
}

func (p *ModuleDirPathCache) stripPrefix(filePath string) string {
	// Check if the file name starts with any of the prefixes:
	for _, prefix := range p.prefixList {
		if strings.HasPrefix(filePath, prefix) {
			// Strip the prefix and return the rest:
			return filePath[len(prefix):]
		}
	}
	// No prefix match, keep the last `keepNDirs` directories:
	pathComp := strings.Split(filePath, "/")
	keepNComps := p.keepNDirs + 1
	if keepNComps < 1 {
		keepNComps = 1
	}
	if keepNComps < len(pathComp) {
		filePath = path.Join(pathComp[len(pathComp)-keepNComps:]...)
	}
	return filePath
}

func (p *ModuleDirPathCache) SetKeepNDirs(n int) {
	p.keepNDirs = n
}

var moduleDirPathCache = &ModuleDirPathCache{
	prefixList: []string{},
	keepNDirs:  1, // typically the last directory is the package
}

// Add the prefix based on the caller's stack, going back `upNDirs` directories
// using the caller's file path. The prefix is added to the list of prefixes to
// be stripped from the file path when logging. The skip parameter is the
// number of stack frames to skip and it is needed when this function is called
// via exported interface, since the latter adds an extra frame.
func AddCallerSrcPathPrefixToLogger(upNDirs int, skip int) error {
	skip += 1 // skip this function
	_, file, _, ok := runtime.Caller(skip)
	if !ok {
		return fmt.Errorf("cannot determine source root: runtime.Caller(%d) failed", skip)
	}
	prefix := path.Dir(file)
	for i := 0; i < upNDirs; i++ {
		prefix = path.Dir(prefix)
	}
	// The prefix should end with a slash, so that it matches a complete path
	// from a file name starting with it (e.g. "/path/to/module/" will match
	// "/path/to/module/pkg/file.go" but not "/path/to/module2/pkg/file.go")
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	moduleDirPathCache.addPrefix(prefix)
	return nil
}

// Maintain a cache for caller PC -> (file:line#, function) to speed up the
// formatting:
type LogFuncFilePair struct {
	function string
	file     string
}

type LogFuncFileCache struct {
	m             *sync.Mutex
	funcFileCache map[uintptr]*LogFuncFilePair
}

// Return the function name and filename:line# info from the frame. The filename is
// relative to the source root dir.
func (c *LogFuncFileCache) LogCallerPrettyfier(f *runtime.Frame) (function string, file string) {
	c.m.Lock()
	defer c.m.Unlock()
	funcFile := c.funcFileCache[f.PC]
	if funcFile == nil {
		funcFile = &LogFuncFilePair{
			"", //f.Function,
			fmt.Sprintf("%s:%d", moduleDirPathCache.stripPrefix(f.File), f.Line),
		}
		c.funcFileCache[f.PC] = funcFile
	}
	return funcFile.function, funcFile.file
}

var logFunctionFileCache = &LogFuncFileCache{
	m:             &sync.Mutex{},
	funcFileCache: make(map[uintptr]*LogFuncFilePair),
}

var LogFieldKeySortOrder = map[string]int{
	// The desired order is time, level, file, func, other fields sorted
	// alphabetically and msg. Use negative numbers for the fields preceding
	// `other' to capitalize on the fact that any of the latter will return 0 at
	// lookup.
	logrus.FieldKeyTime:         -5,
	logrus.FieldKeyLevel:        -4,
	LOGGER_COMPONENT_FIELD_NAME: -3,
	logrus.FieldKeyFile:         -2,
	logrus.FieldKeyFunc:         -1,
	logrus.FieldKeyMsg:          1,
}

type LogFieldKeySortable struct {
	keys []string
}

func (d *LogFieldKeySortable) Len() int {
	return len(d.keys)
}

func (d *LogFieldKeySortable) Less(i, j int) bool {
	key_i, key_j := d.keys[i], d.keys[j]
	order_i, order_j := LogFieldKeySortOrder[key_i], LogFieldKeySortOrder[key_j]
	if order_i != 0 || order_j != 0 {
		return order_i < order_j
	}
	return strings.Compare(key_i, key_j) == -1
}

func (d *LogFieldKeySortable) Swap(i, j int) {
	d.keys[i], d.keys[j] = d.keys[j], d.keys[i]
}

func LogSortFieldKeys(keys []string) {
	sort.Sort(&LogFieldKeySortable{keys})
}

var LogTextFormatter = &logrus.TextFormatter{
	DisableColors:    true,
	DisableQuote:     false,
	FullTimestamp:    true,
	TimestampFormat:  LOGGER_TIMESTAMP_FORMAT,
	CallerPrettyfier: logFunctionFileCache.LogCallerPrettyfier,
	DisableSorting:   false,
	SortingFunc:      LogSortFieldKeys,
}

var LogJsonFormatter = &logrus.JSONFormatter{
	TimestampFormat:  LOGGER_TIMESTAMP_FORMAT,
	CallerPrettyfier: logFunctionFileCache.LogCallerPrettyfier,
}

var RootLogger = &CollectableLogger{
	Logger: logrus.Logger{
		Out: os.Stderr,
		//Hooks:        make(logrus.LevelHooks),
		Formatter:    LogTextFormatter,
		Level:        LOGGER_DEFAULT_LEVEL,
		ReportCaller: true,
	},
}

// Public access to the root logger, needed for testing:
func GetRootLogger() *CollectableLogger { return RootLogger }

func GetLogLevelNames() []string {
	levelNames := make([]string, len(logrus.AllLevels))
	for i, level := range logrus.AllLevels {
		levelNames[i] = level.String()
	}
	return levelNames
}

func init() {
	// Add the default prefix for the current module, which is 2 dirs up from
	// here. Do not skip extra frames, since this is a direct call, not via
	// an exported interface.
	AddCallerSrcPathPrefixToLogger(2, 0)
}

// Set the logger based on config overridden by command line args, if the latter
// were used:
func SetLogger(logCfg *LoggerConfig) error {
	if logCfg == nil {
		logCfg = DefaultLoggerConfig()
	}

	levelName := logCfg.Level
	if levelName != "" {
		level, err := logrus.ParseLevel(levelName)
		if err != nil {
			return err
		}
		RootLogger.SetLevel(level)
	}

	if logCfg.UseJson {
		RootLogger.SetFormatter(LogJsonFormatter)
	} else {
		RootLogger.SetFormatter(LogTextFormatter)
	}

	RootLogger.SetReportCaller(!logCfg.DisableSrcFile)

	switch logFile := logCfg.LogFile; logFile {
	case "stderr":
		RootLogger.SetOutput(os.Stderr)
	case "stdout":
		RootLogger.SetOutput(os.Stdout)
	case "":
	default:
		// Create log dir as needed:
		logDir := path.Dir(logCfg.LogFile)
		_, err := os.Stat(logDir)
		if err != nil {
			err = os.MkdirAll(logDir, os.ModePerm)
			if err != nil {
				return err
			}
		}
		// Check if the log file exists, in which case force rotate it before
		// the 1st use:
		_, err = os.Stat(logCfg.LogFile)
		forceRotate := err == nil
		logFile := &lumberjack.Logger{
			Filename:   logCfg.LogFile,
			MaxSize:    logCfg.LogFileMaxSizeMB,
			MaxBackups: logCfg.LogFileMaxBackupNum,
		}
		if forceRotate {
			err := logFile.Rotate()
			if err != nil {
				return err
			}
		}
		RootLogger.SetOutput(logFile)
	}

	return nil
}

func NewCompLogger(compName string) *logrus.Entry {
	return RootLogger.WithField(LOGGER_COMPONENT_FIELD_NAME, compName)
}
