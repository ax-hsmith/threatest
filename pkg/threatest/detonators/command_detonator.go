package detonators

import (
	"github.com/hashicorp/go-uuid"
	"github.com/datadog/threatest/pkg/threatest/logging"
	log "github.com/sirupsen/logrus"
)

//TODO probably not a full struct needed
type OSLayerAttackTechnique struct {
	Command string
}

type CommandDetonator interface {
	RunCommand(command string, detonationID string, log *log.Entry) error
}

type CommandDetonatorImpl struct {
	Detonator CommandDetonator
	Technique *OSLayerAttackTechnique
}

func NewCommandDetonator(detonator CommandDetonator, command string) *CommandDetonatorImpl {
	return &CommandDetonatorImpl{
		Detonator: detonator,
		Technique: &OSLayerAttackTechnique{Command: command},
	}
}

func (m *CommandDetonatorImpl) Detonate() (string, error) {
	id, _ := uuid.GenerateUUID()
	logger := logging.NewScenarioLogger(id)
	err := m.Detonator.RunCommand(m.Technique.Command, id, logger)
	return id, err
}
