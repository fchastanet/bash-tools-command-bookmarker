package services

import (
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/pkg/db"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
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

func (s *DBService) SaveCommand(command *models.Command) error {
	// Use Exec instead of Query for INSERT statements
	result, err := s.dbAdapter.GetDB().Exec(
		`INSERT INTO command (
			title, description, script, status,
			lint_issues, lint_status, elapsed,
			creation_datetime, modification_datetime
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		command.Title, command.Description, command.Script, string(command.Status),
		command.LintIssues, string(command.LintStatus), command.Elapsed,
		command.CreationDatetime.Format(time.DateTime), command.ModificationDatetime.Format(time.DateTime),
	)
	if err != nil {
		return err
	}
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		slog.Error("Error retrieving last insert ID", "error", err)
		return err
	}
	command.ID = resource.ID(lastInsertID)
	return nil
}

func (s *DBService) DuplicateCommand(commandID resource.ID, status models.CommandStatus) (resource.ID, error) {
	// Use Exec instead of Query for INSERT statements
	result, err := s.dbAdapter.GetDB().Exec(
		`INSERT INTO command (
			title, description, script, status,
			lint_issues, lint_status, elapsed,
			creation_datetime, modification_datetime
		) SELECT
			title, description, script, ?,
			lint_issues, lint_status, elapsed,
			creation_datetime, ?
		FROM command WHERE id = ?`,
		status,
		time.Now().Format(time.DateTime),
		commandID,
	)
	if err != nil {
		return -1, err
	}
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		slog.Error("Error retrieving last insert ID", "error", err)
		return -1, err
	}
	return resource.ID(lastInsertID), nil
}

// GetCommandByID retrieves a command by its database ID
func (s *DBService) GetCommandByID(id resource.ID) (*models.Command, error) {
	slog.Debug("Retrieving command by id from database", "id", id)
	// Use QueryRow for single row retrieval
	row := s.dbAdapter.GetDB().QueryRow(
		`SELECT id, title, description, script, status,
			lint_issues, lint_status, elapsed,
			creation_datetime, modification_datetime
			FROM command WHERE id = ? LIMIT 1`,
		id,
	)
	if row == nil {
		slog.Debug("No command found in database", "id", id)
		return nil, nil
	}
	return s.getCommandFromRow(row)
}

func (s *DBService) GetCommandByScript(script string) (*models.Command, error) {
	slog.Debug("Retrieving command by script from database", "script", script)
	// Use QueryRow for single row retrieval
	row := s.dbAdapter.GetDB().QueryRow(
		`SELECT id, title, description, script, status,
			lint_issues, lint_status, elapsed,
			creation_datetime, modification_datetime
			FROM command WHERE script = ? LIMIT 1`,
		script,
	)
	if row == nil {
		slog.Debug("No command found in database", "script", script)
		return nil, nil
	}

	return s.getCommandFromRow(row)
}

func (*DBService) getCommandFromRow(row *sql.Row) (*models.Command, error) {
	var command models.Command
	var creationDateStr string
	var modificationDateStr string

	err := row.Scan(
		&command.ID,
		&command.Title,
		&command.Description,
		&command.Script,
		&command.Status,
		&command.LintIssues,
		&command.LintStatus,
		&command.Elapsed,
		&creationDateStr,
		&modificationDateStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		// Handle other scan errors
		slog.Error("Error scanning command from database", "error", err)
		return nil, err
	}

	command.CreationDatetime, err = time.Parse(time.DateTime, creationDateStr)
	if err != nil {
		return nil, err
	}

	command.ModificationDatetime, err = time.Parse(time.DateTime, modificationDateStr)
	if err != nil {
		return nil, err
	}
	return &command, nil
}

func (s *DBService) GetMaxCommandTimestamp() (time.Time, error) {
	var maxTimestampStr string
	var maxTimestamp time.Time

	// Query for the maximum creation_datetime
	row := s.dbAdapter.GetDB().QueryRow("SELECT IFNULL(MAX(creation_datetime), '1970-01-01 00:00:00') FROM command")
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

// GetCommands retrieves commands from the database, optionally filtered by status
func (s *DBService) GetCommands(statuses ...models.CommandStatus) ([]*models.Command, error) {
	var commands []*models.Command
	var creationDateStr string
	var modificationDateStr string
	var query string
	var args []interface{}

	// Base query
	query = `SELECT id, title, description, script, status,
		lint_issues, lint_status, elapsed,
		creation_datetime, modification_datetime
		FROM command`

	// Add status filter if provided
	if len(statuses) > 0 {
		query += " WHERE status IN ("
		placeholders := make([]string, len(statuses))
		for i, status := range statuses {
			placeholders[i] = "?"
			args = append(args, string(status))
		}
		query += strings.Join(placeholders, ", ") + ")"
	}

	// Execute the query
	rows, err := s.dbAdapter.GetDB().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		command := models.Command{
			ID:                   0,
			Title:                "",
			Description:          "",
			Script:               "",
			Status:               "",
			LintIssues:           "",
			LintStatus:           "",
			Elapsed:              0,
			CreationDatetime:     time.Time{},
			ModificationDatetime: time.Time{},
		}
		err := rows.Scan(
			&command.ID,
			&command.Title,
			&command.Description,
			&command.Script,
			&command.Status,
			&command.LintIssues,
			&command.LintStatus,
			&command.Elapsed,
			&creationDateStr,
			&modificationDateStr,
		)
		if err != nil {
			return nil, err
		}

		command.CreationDatetime, err = time.Parse(time.DateTime, creationDateStr)
		if err != nil {
			return nil, err
		}

		command.ModificationDatetime, err = time.Parse(time.DateTime, modificationDateStr)
		if err != nil {
			return nil, err
		}
		commands = append(commands, &command)
	}
	return commands, nil
}

// UpdateCommand updates an existing command in the database
func (s *DBService) UpdateCommand(command *models.Command) error {
	slog.Debug("Updating command in database", "command", command)
	// Use Exec for UPDATE statements
	_, err := s.dbAdapter.GetDB().Exec(`UPDATE command
		SET title = ?, description = ?, script = ?,
		status = ?, lint_issues = ?, lint_status = ?,
		elapsed = ?, modification_datetime = ?
		WHERE id = ?`,
		command.Title, command.Description, command.Script,
		string(command.Status), command.LintIssues, string(command.LintStatus),
		command.Elapsed, time.Now().Format(time.DateTime), command.ID,
	)
	if err != nil {
		slog.Error("Error updating command in database", "id", command.ID, "error", err)
		return err
	}
	slog.Info("Command updated successfully", "id", command.ID)
	return nil
}

// GetCommandCountsByStatus retrieves a count of commands grouped by status directly from the database
func (s *DBService) GetCommandCountsByStatus() (map[models.CommandStatus]int, error) {
	// Use SQL GROUP BY to count by status directly in the database
	query := `SELECT status, COUNT(*) as count FROM command GROUP BY status`

	rows, err := s.dbAdapter.GetDB().Query(query)
	if err != nil {
		slog.Error("Error querying command counts by status", "error", err)
		return nil, err
	}
	defer rows.Close()

	counts := make(map[models.CommandStatus]int)

	for rows.Next() {
		var statusStr string
		var count int

		if err := rows.Scan(&statusStr, &count); err != nil {
			slog.Error("Error scanning command count row", "error", err)
			return nil, err
		}

		status := models.CommandStatus(statusStr)
		counts[status] = count
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating command count rows", "error", err)
		return nil, err
	}

	return counts, nil
}
