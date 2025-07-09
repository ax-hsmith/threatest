package logging

import (
	"fmt"
	logrus "github.com/sirupsen/logrus"
)

// ScenarioFormatter wraps the default formatter and adds detonation ID prefix
type ScenarioFormatter struct {
	logrus.Formatter
}

// Format formats log entries with detonation ID prefix when present
func (f *ScenarioFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Check if detonation_id field exists - if so, modify the entry before formatting
	if detonationID, exists := entry.Data["detonation_id"]; exists {
		// Create a copy of the entry to avoid modifying the original
		entryCopy := *entry
		entryCopy.Data = make(logrus.Fields)
		for k, v := range entry.Data {
			if k != "detonation_id" {
				entryCopy.Data[k] = v
			}
		}
		// Prefix the message with the detonation ID
		entryCopy.Message = fmt.Sprintf("[%s] %s", detonationID, entry.Message)
		return f.Formatter.Format(&entryCopy)
	}

	return f.Formatter.Format(entry)
}

// init sets up the custom formatter when the package is imported
func init() {
	// Only set if no custom formatter is already set
	if _, ok := logrus.StandardLogger().Formatter.(*ScenarioFormatter); !ok {
		logrus.SetFormatter(&ScenarioFormatter{
			Formatter: logrus.StandardLogger().Formatter,
		})
	}
}

// NewScenarioLogger creates a new logger with the detonation ID field
func NewScenarioLogger(detonationID string) *logrus.Entry {
	return logrus.WithField("detonation_id", detonationID)
}