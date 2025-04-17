package services

import (
	"time"

	"github.com/fchastanet/shell-command-bookmarker/app/processors"
	"github.com/fchastanet/shell-command-bookmarker/internal/db"
)

type CommandStatus string

const (
	CommandStatusImported   CommandStatus = "IMPORTED"
	CommandStatusBookmarked CommandStatus = "BOOKMARKED"
)

type DBService struct {
	dbAdapter  db.Adapter
	dbPath     string
	schemaPath string
}

func NewDBService(
	dbPath string,
	schemaPath string,
) *DBService {
	return &DBService{
		dbAdapter:  db.NewSQLiteAdapter(dbPath, schemaPath),
		dbPath:     dbPath,
		schemaPath: schemaPath,
	}
}

func (s *DBService) Open() error {
	return s.dbAdapter.Open()
}

func (s *DBService) Close() error {
	return s.dbAdapter.Close()
}

func (s *DBService) GetDBAdapter() db.Adapter {
	return s.dbAdapter
}

func (s *DBService) SaveCommand(command processors.HistoryCommand) error {
	date := command.Timestamp.Format(time.DateTime)
	// Use Exec instead of Query for INSERT statements
	_, err := s.dbAdapter.GetDB().Exec(
		"INSERT INTO command (title, description, script, elapsed, creation_datetime, modification_datetime, status) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?)",
		"", "",
		command.Command, command.Elapsed, date, date,
		string(CommandStatusImported),
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *DBService) GetMaxCommandTimestamp() (time.Time, error) {
	var maxTimestampStr string
	var maxTimestamp time.Time

	// Query for the maximum creation_datetime
	row := s.dbAdapter.GetDB().QueryRow("SELECT MAX(creation_datetime) FROM command")
	err := row.Scan(&maxTimestampStr)
	if err != nil {
		// Handle case where table might be empty or other scan errors
		// If the table is empty, Scan might return sql.ErrNoRows, depending on the driver and MAX behavior with NULLs.
		// We might want to return time.Zero or a specific error in that case.
		// For simplicity, returning zero time and the error for now.
		return time.Time{}, err
	}

	// Parse the timestamp string into time.Time
	// Assuming the format stored is time.DateTime ("2006-01-02 15:04:05")
	maxTimestamp, err = time.Parse(time.DateTime, maxTimestampStr)
	if err != nil {
		return time.Time{}, err
	}

	return maxTimestamp, nil
}
