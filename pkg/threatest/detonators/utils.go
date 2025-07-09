package detonators

import (
	"fmt"
	"strings"
	"gopkg.in/alessio/shellescape.v1"
)

func FormatCommand(rawCommand string, detonationUuid string) string {
	// Replace newlines with semicolons to make multi-line commands executable
	singleLineCommand := strings.ReplaceAll(strings.TrimSpace(rawCommand), "\n", "; ")
	
	// Use environment variable for correlation instead of copying bash
	// This avoids SIP issues on macOS while still providing something to correlate on
	return fmt.Sprintf(
		`export THREATEST_DETONATION_ID=%[1]s; bash -c %[2]s`,
		detonationUuid, shellescape.Quote(singleLineCommand),
	)
}
