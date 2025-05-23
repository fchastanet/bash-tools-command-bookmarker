package processors

import (
	"bufio"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const ExtendedCommandPrefixLen = 2

// controlCharsRegexp is a regex to match control characters
var controlCharsRegexp = regexp.MustCompile(`[^\x00-\x7F]`)

type HistoryIngestor struct {
	// parsedCmdCount is the number of commands parsed from the history file
	parsedCmdCount int
	// importedCmdCount is the number of commands successfully imported
	// into the database
	importedCmdCount int
	// skippedCmdCount is the number of commands skipped
	skippedCmdCount int
	// errorCmdCount is the number of commands that has lint errors
	// or other errors
	errorCmdCount int
	// alreadyExistsCmdCount is the number of commands that already exist
	// in the database and were not imported
	alreadyExistsCmdCount int
	// filteredOutCmdCount is the number of commands that were filtered out
	// by the ignore lines regexps
	filteredOutCmdCount int
}

// NewHistoryIngestor creates a new HistoryIngestor instance
func NewHistoryIngestor() *HistoryIngestor {
	return &HistoryIngestor{
		parsedCmdCount:        0,
		importedCmdCount:      0,
		skippedCmdCount:       0,
		errorCmdCount:         0,
		alreadyExistsCmdCount: 0,
		filteredOutCmdCount:   0,
	}
}

// HistoryCommand represents a single command entry from bash history
type HistoryCommand struct {
	Timestamp     time.Time
	Command       string
	Elapsed       int // elapsed time in seconds
	ParseFinished bool
}

// timestampFieldsCount is the number of fields in the extended format
const timestampFieldsCount = 2

func (*HistoryIngestor) OpenHistoryFile(historyFilePath string) (*os.File, error) {
	// If no specific path is provided, use the default ~/.bash_history
	if historyFilePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err // Propagate error getting home dir
		}
		historyFilePath = filepath.Join(homeDir, ".bash_history")
	}

	file, err := os.Open(historyFilePath) // #nosec G304
	if err != nil {
		return nil, err // Propagate file open error
	}

	return file, nil
}

// ParseBashHistory reads and parses the bash history file
// It supports both simple format (just commands) and extended format (`: start:elapsed;command`)
// It handles multi-line commands indicated by a trailing backslash '\'.
func (h *HistoryIngestor) ParseBashHistory(
	historyFilePath string,
	fromTimestamp time.Time,
	callback func(HistoryCommand) (CommandImportedStatus, error),
) error {
	file, err := h.OpenHistoryFile(historyFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var commandBuilder strings.Builder
	var importStatus CommandImportedStatus
	var currentCommand *HistoryCommand // Pointer to track the command being built

	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		// Skip empty lines only if not currently building a multi-line command
		if line == "" && currentCommand == nil {
			continue
		}

		h.processHistoryLine(line, &commandBuilder, &currentCommand)

		if importStatus, err = h.handleCommand(
			historyFilePath, lineNumber, fromTimestamp, currentCommand, callback,
		); err != nil {
			// Stop processing if the callback returns an error
			h.updateStats(importStatus)
			return err // Propagate callback error
		}
		h.updateStats(importStatus)

		if currentCommand.ParseFinished {
			// If the command is fully parsed, we can reset the command builder
			currentCommand = nil
			commandBuilder.Reset()
		}
	}

	// Handle case where the file ends while building a multi-line command
	// (e.g., the last command in the file is multi-line without a final newline)
	if currentCommand != nil {
		// Finalize the command built so far
		currentCommand.ParseFinished = true // Mark as fully parsed
		currentCommand.Command = commandBuilder.String()

		if importStatus, err = h.handleCommand(
			historyFilePath, lineNumber, fromTimestamp, currentCommand, callback,
		); err != nil {
			h.updateStats(importStatus)
			return err // Propagate callback error
		}
		h.updateStats(importStatus)
	}

	if err := scanner.Err(); err != nil {
		return err // Propagate scanner error
	}
	slog.Debug(
		"History ingestion stats",
		"historyFilePath", historyFilePath,
		"fromTimestamp", fromTimestamp,
		"parsedCmdCount", h.parsedCmdCount,
		"importedCmdCount", h.importedCmdCount,
		"skippedCmdCount", h.skippedCmdCount,
		"errorCmdCount", h.errorCmdCount,
		"alreadyExistsCmdCount", h.alreadyExistsCmdCount,
		"filteredOutCmdCount", h.filteredOutCmdCount,
	)

	return nil
}

// updateStats updates the statistics based on the import status
func (h *HistoryIngestor) updateStats(importStatus CommandImportedStatus) {
	if importStatus != CommandImportedStatusInProgress {
		h.parsedCmdCount++
	}
	// Update counts based on the import status
	switch importStatus {
	case CommandImportedStatusNew:
		h.importedCmdCount++
	case CommandImportedStatusError:
		h.errorCmdCount++
	case CommandImportedStatusFilteredOut:
		h.filteredOutCmdCount++
	case CommandImportedStatusAlreadyExists:
		h.alreadyExistsCmdCount++
	case CommandImportedStatusSkipped:
		h.skippedCmdCount++
	case CommandImportedStatusInProgress:
		// Do nothing for in-progress status
	default:
		slog.Error("Unknown command import status", "importStatus", importStatus)
	}
}

type CommandImportedStatus int

const (
	CommandImportedStatusNew CommandImportedStatus = iota
	CommandImportedStatusSkipped
	CommandImportedStatusFilteredOut
	CommandImportedStatusAlreadyExists
	CommandImportedStatusError
	CommandImportedStatusInProgress
)

func (*HistoryIngestor) handleCommand(
	historyFilePath string,
	lineNumber int,
	fromTimestamp time.Time,
	cmd *HistoryCommand,
	callback func(HistoryCommand) (CommandImportedStatus, error),
) (CommandImportedStatus, error) {
	if !cmd.ParseFinished {
		return CommandImportedStatusInProgress, nil // Skip if the command is not fully parsed
	}
	if strings.TrimSpace(cmd.Command) == "" {
		return CommandImportedStatusSkipped, nil // Skip empty commands
	}
	importedStatus := CommandImportedStatusSkipped
	var err error
	if cmd.Timestamp.IsZero() || cmd.Timestamp.After(fromTimestamp) {
		if importedStatus, err = callback(*cmd); err != nil {
			return importedStatus, err
		}
	} else {
		slog.Debug(
			"Skipping command due to timestamp",
			"historyFilePath", historyFilePath,
			"timestamp", cmd.Timestamp,
			"fromTimestamp", fromTimestamp,
			"lineNumber", lineNumber,
		)
	}
	return importedStatus, nil
}

// processHistoryLine handles the logic for a single line from the history file.
// It updates the currentCommand being built and returns true if a command is completed.
// The currentCommand pointer (**cmd) allows modification of the caller's currentCommand variable.
func (*HistoryIngestor) processHistoryLine(
	line string, commandBuilder *strings.Builder, cmd **HistoryCommand,
) {
	currentCommand := *cmd // Dereference to work with the actual *HistoryCommand

	part := line
	if currentCommand == nil {
		// Start of a potential new command
		var ts time.Time
		var el int
		ts, el, part, _ = parseFirstHistoryLine(line)
		// Initialize the command being built
		currentCommand = &HistoryCommand{
			Timestamp:     ts,
			Elapsed:       el,
			Command:       "", // Command string set later
			ParseFinished: false,
		}
		*cmd = currentCommand // Update the caller's pointer
	}

	if strings.HasSuffix(part, "\\") {
		// Start of a multi-line command
		part = strings.TrimSuffix(part, "\\") // Remove the trailing backslash
		part = strings.TrimRight(part, " \t") // Remove trailing spaces
		commandBuilder.WriteString(part)
		commandBuilder.WriteString("\n") // Add newline separator
		// command Not completed yet
	} else {
		commandBuilder.WriteString(part) // Append the final line
		currentCommand.Command = cleanCommand(commandBuilder.String())
		currentCommand.ParseFinished = true // Mark command as fully parsed
	}
}

func cleanCommand(command string) string {
	// Remove control characters
	command = removeControlCharacters(command)
	// Remove empty lines at begin/end
	command = strings.TrimLeft(command, "\n")
	command = strings.TrimRight(command, "\n")
	// Remove leading and trailing whitespace
	command = strings.TrimSpace(command)
	return command
}

// removeControlCharacters removes control characters from the command string
func removeControlCharacters(command string) string {
	return controlCharsRegexp.ReplaceAllString(command, "")
}

// parseFirstHistoryLine parses the first line of a command entry.
// It returns the timestamp, elapsed time, the initial command part,
// and a boolean indicating if it's in extended format.
// Extended format expected: ": <unix_timestamp>:<elapsed_seconds>;<command>"
func parseFirstHistoryLine(line string) (timestamp time.Time, elapsed int, commandPart string, isExtendedFormat bool) {
	timestamp = time.Now().UTC() // Default for simple format or errors
	elapsed = 0                  // Default
	commandPart = line           // Default: treat the whole line as the command initially
	isExtendedFormat = false

	// Check for extended format prefix ": "
	if !strings.HasPrefix(line, ": ") {
		// Not extended format, return defaults
		return timestamp, elapsed, commandPart, isExtendedFormat
	}

	// Potential extended format: ": start:elapsed;command"
	semicolonPos := strings.Index(line, ";")
	// Ensure semicolon exists and is after the potential timestamp info
	if semicolonPos <= ExtendedCommandPrefixLen { // Needs space for ": T:E;" at minimum
		// Malformed or not the expected extended format structure, return defaults
		return timestamp, elapsed, commandPart, isExtendedFormat
	}

	// Extract timestamps part and potential command part
	timestampsPart := line[ExtendedCommandPrefixLen:semicolonPos] // Skip ": "
	potentialCommandPart := line[semicolonPos+1:]

	// Try parsing timestamps
	parsedTimestamp, parsedElapsed, err := ParseTimestamp(timestampsPart)
	if err == nil {
		// Successfully parsed extended format
		timestamp = parsedTimestamp
		elapsed = parsedElapsed
		commandPart = potentialCommandPart // Update commandPart only on success
		isExtendedFormat = true
	}
	// If parsing failed, the defaults (entire line as commandPart, etc.) are returned.

	return timestamp, elapsed, commandPart, isExtendedFormat
}

var (
	errInvalidTimestampFormat = errors.New("invalid timestamp format")
	errInvalidTimestamp       = errors.New("invalid timestamp")
	errInvalidElapsedFormat   = errors.New("invalid elapsed format")
)

func ParseTimestamp(timestampsPart string) (time.Time, int, error) {
	// Split timestamps part by colon
	timestampFields := strings.Split(timestampsPart, ":")
	if len(timestampFields) != timestampFieldsCount {
		// Invalid format
		return time.Time{}, 0, errInvalidTimestampFormat
	}

	// Parse start timestamp
	startTimestamp, err := strconv.ParseInt(timestampFields[0], 10, 64)
	if err != nil {
		// Parsing failed
		return time.Time{}, 0, errInvalidTimestamp
	}

	// Parse elapsed time
	elapsed, err := strconv.Atoi(timestampFields[1])
	if err != nil {
		// Parsing failed
		return convertUnixToUTC(startTimestamp), 0, errInvalidElapsedFormat
	}

	return convertUnixToUTC(startTimestamp), elapsed, nil
}

// convertUnixToUTC converts a Unix timestamp in the user's timezone to UTC
func convertUnixToUTC(unixTimestamp int64) time.Time {
	return time.Unix(unixTimestamp, 0).UTC()
}
