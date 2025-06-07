package tui

import (
	"fmt"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Define constants to replace magic numbers
const (
	// BytesInMegabyte is the number of bytes in a megabyte (1024 * 1024)
	BytesInMegabyte = 1024 * 1024
)

func CmdHandler(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

type ErrorMsg error

func ReportError(err error) tea.Cmd {
	return CmdHandler(ErrorMsg(err))
}

type InfoMsg string

func ReportInfo(msg string, args ...any) tea.Cmd {
	return CmdHandler(InfoMsg(fmt.Sprintf(msg, args...)))
}

// MemoryStatsMsg is a message containing memory usage statistics
type MemoryStatsMsg struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	NumGC      uint32
}

// GetMemoryStats returns a command that sends memory statistics
func GetMemoryStats() tea.Cmd {
	return func() tea.Msg {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return MemoryStatsMsg{
			Alloc:      m.Alloc / BytesInMegabyte,      // MB
			TotalAlloc: m.TotalAlloc / BytesInMegabyte, // MB
			Sys:        m.Sys / BytesInMegabyte,        // MB
			NumGC:      m.NumGC,
		}
	}
}

// PerformanceMonitorStartMsg is a message to start the performance monitor
type PerformanceMonitorStartMsg struct{}

// PerformanceMonitorStopMsg is a message to stop the performance monitor
type PerformanceMonitorStopMsg struct{}

// StartPerformanceMonitor starts periodic memory statistics monitoring
func StartPerformanceMonitor(_ time.Duration) tea.Cmd {
	return func() tea.Msg {
		return PerformanceMonitorStartMsg{}
	}
}

// StopPerformanceMonitor stops the performance monitor
func StopPerformanceMonitor() tea.Cmd {
	return func() tea.Msg {
		return PerformanceMonitorStopMsg{}
	}
}

// PerformanceMonitorTick generates periodic performance monitoring commands
func PerformanceMonitorTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return GetMemoryStats()()
	})
}

// FilterFocusReqMsg is a request to focus the filter widget.
type FilterFocusReqMsg struct{}

// FilterValidateMsg is a request to validate the filter widget. It is not
// acknowledged.
type FilterValidateMsg struct{}

// FilterCloseMsg is a request to close the filter widget. It is not
// acknowledged.
type FilterCloseMsg struct{}

// FilterKeyMsg is a key entered by the user into the filter widget
type FilterKeyMsg tea.KeyMsg

// DummyMsg can be used to indicate that Update method has treated a message
// but does not need to return a command.
type DummyMsg struct{}

func GetDummyCmd() tea.Cmd {
	return func() tea.Msg {
		return DummyMsg{}
	}
}
