package services

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/davecgh/go-spew/spew"
)

const WriteFileMode = 0o644

type LoggerService struct {
	logFileHandler  io.WriteCloser
	dumpFileHandler io.WriteCloser
	debug           bool
}

func NewLoggerService(debugMode bool) *LoggerService {
	return &LoggerService{
		debug:           debugMode,
		logFileHandler:  nil,
		dumpFileHandler: nil,
	}
}

func (s *LoggerService) Init() error {
	var err error
	s.logFileHandler, err = openFileInWriteMode("logs/tui.log")
	if err != nil {
		return err
	}

	level := slog.LevelError
	if s.debug {
		level = slog.LevelDebug
	}

	if err := s.initLogger(level); err != nil {
		return err
	}

	if s.debug {
		s.dumpFileHandler, err = openFileInWriteMode("logs/dump.log")
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *LoggerService) LogTeaMsg(msg tea.Msg) {
	if s.dumpFileHandler == nil {
		return
	}
	spew.Fdump(s.dumpFileHandler, msg)
}

// EnhancedLogTeaMsg provides detailed logging of tea messages, with special handling for key events
func (s *LoggerService) EnhancedLogTeaMsg(msg tea.Msg) {
	// Add a recover function to catch panics during logging
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic during message logging",
				"error", r,
				"stack", debug.Stack())
		}
	}()

	// Before using spew.Dump on an interface value, check if it's valid
	if msg == nil {
		slog.Warn("Attempted to log nil message")
		return
	}

	// Use type assertions to check specific problematic types
	switch v := msg.(type) {
	case error:
		// Handle error type safely
		slog.Debug("Message is an error", "error", v.Error())
	default:
		// For other types, use a safer logging approach
		slog.Debug("Tea message",
			"type", reflect.TypeOf(msg).String(),
			"value", fmt.Sprintf("%+v", msg))

		// Only use spew.Dump if you really need the detailed output
		// and consider wrapping it in a safe logging function
		s.safeDump(msg)
	}
}

// Add a helper method to safely dump values
func (s *LoggerService) safeDump(v any) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic during value dump",
				"error", r,
				"valueType", reflect.TypeOf(v))
		}
	}()

	// Optional: Check if the value is a pointer and if it's nil
	if reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil() {
		slog.Debug("Attempted to dump nil pointer")
		return
	}

	if s.dumpFileHandler == nil {
		slog.Warn("Dump file handler is not initialized, skipping dump")
		return
	}
	// Now safely use spew.Dump
	spew.Fdump(s.dumpFileHandler, v)
}

func (s *LoggerService) Close() error {
	if s.logFileHandler != nil {
		if err := s.logFileHandler.Close(); err != nil {
			slog.Error("Error closing log file handler", "error", err)
			return err
		}
	}
	if s.dumpFileHandler != nil {
		if err := s.dumpFileHandler.Close(); err != nil {
			slog.Error("Error closing dump file handler", "error", err)
			return err
		}
	}
	return nil
}

func openFileInWriteMode(filePath string) (io.WriteCloser, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, WriteFileMode) // #nosec G304
	if err != nil {
		slog.Error("Error opening debug log file", "error", err)
		return nil, err
	}
	return file, nil
}

func (s *LoggerService) initLogger(level slog.Level) error {
	var err error
	s.logFileHandler, err = openFileInWriteMode("logs/error.log")
	if err != nil {
		return err
	}
	slog.SetLogLoggerLevel(level)
	opts := &slog.HandlerOptions{
		AddSource:   level == slog.LevelDebug,
		Level:       level,
		ReplaceAttr: nil,
	}
	handler := slog.NewTextHandler(s.logFileHandler, opts)

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return nil
}
