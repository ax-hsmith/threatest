package detonators

import (
	"os/exec"
	log "github.com/sirupsen/logrus"
)

type LocalCommandExecutor struct{}

func (m *LocalCommandExecutor) RunCommand(command string, detonationID string, logger *log.Entry) error {
	logger.Infof("Executing %s", command)
	_, err := exec.Command("bash", "-c", FormatCommand(command, detonationID)).Output()
	return err
}
