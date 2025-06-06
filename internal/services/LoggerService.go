package services

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/davecgh/go-spew/spew"
)

const WriteFileMode = 0o644

type LoggerService struct {
	logFileHandler  io.WriteCloser
	dumpFileHandler io.WriteCloser
	debug           bool
}

func NewLoggerService(debug bool) *LoggerService {
	return &LoggerService{
		debug:           debug,
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
	if s.dumpFileHandler == nil {
		return
	}

	// Special handling for key messages to make debugging easier
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		fmt.Fprintf(s.dumpFileHandler, "KeyMsg: Type=%v, Runes=%v, String=%q\n",
			keyMsg.Type, keyMsg.Runes, keyMsg.String())
		return
	}

	// For all other message types, use spew for detailed dumps
	spew.Fdump(s.dumpFileHandler, msg)
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
