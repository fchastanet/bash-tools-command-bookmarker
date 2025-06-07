package command

import (
	"fmt"

	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
)

// ErrComposeCommand represents an error when composing a command fails
type ErrComposeCommand struct {
	Err error
}

func (e *ErrComposeCommand) Error() string {
	return fmt.Sprintf("failed to compose command: %v", e.Err)
}

// ErrRestoreCommand represents an error when restoring a command fails
type ErrRestoreCommand struct {
	Err error
}

func (e *ErrRestoreCommand) Error() string {
	return fmt.Sprintf("failed to restore command: %v", e.Err)
}

// ErrSelectionMismatch is returned when selection is not compatible with the operation
type ErrSelectionMismatch struct{}

func (e *ErrSelectionMismatch) Error() string {
	return "selection mismatch: the selected commands do not match the expected criteria for this operation"
}

// ErrNoCommandsSelected is returned when no commands are selected for an operation
type ErrNoCommandsSelected struct{}

func (e *ErrNoCommandsSelected) Error() string {
	return "no commands selected to copy"
}

// ErrClipboardCopyFailed is returned when copying to clipboard fails
type ErrClipboardCopyFailed struct {
	Err error
}

func (e *ErrClipboardCopyFailed) Error() string {
	return "failed to copy to clipboard: " + e.Err.Error()
}

type ErrCommandLoadingFailure struct {
	Err       error
	CommandID resource.ID
}

func (e *ErrCommandLoadingFailure) Error() string {
	return fmt.Sprintf("failed to load command with ID %d: %v", e.CommandID, e.Err)
}
