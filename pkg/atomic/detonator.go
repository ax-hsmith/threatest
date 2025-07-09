package atomic

import (
	"fmt"
	"github.com/datadog/threatest/pkg/threatest/detonators"
	"github.com/datadog/threatest/pkg/threatest/logging"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// ARTDetonator wraps a command detonator with ART-specific functionality
type ARTDetonator struct {
	commandDetonator *detonators.CommandDetonatorImpl
	cleanupCommand   *string
	cleanupExecuted  bool
	test             *Test
	inputs           map[string]string
	executor         detonators.CommandDetonator
	detonationID     string
	log              *log.Entry
}

// NewARTDetonator creates a new ART detonator with cleanup and prerequisite support
func NewARTDetonator(executor detonators.CommandDetonator, command string, cleanupCommand *string, test *Test, inputs map[string]string) *ARTDetonator {
	art := &ARTDetonator{
		commandDetonator: detonators.NewCommandDetonator(executor, command),
		cleanupCommand:   cleanupCommand,
		test:             test,
		inputs:           inputs,
		executor:         executor,
	}
	
	// Set up signal handler for cleanup on interrupt
	if cleanupCommand != nil {
		art.setupSignalHandler()
	}
	
	return art
}

// Detonate executes the ART test command after checking prerequisites
func (a *ARTDetonator) Detonate() (string, error) {
	// Check and install prerequisites if needed
	if err := a.checkPrerequisites(); err != nil {
		return "", err
	}
	
	detonationID, err := a.commandDetonator.Detonate()
	if err != nil {
		return "", err
	}
	
	// Store the detonation ID and create scenario logger for cleanup
	a.detonationID = detonationID
	a.log = logging.NewScenarioLogger(detonationID)
	return detonationID, nil
}

// Cleanup executes the cleanup command if it exists and hasn't been run yet
func (a *ARTDetonator) Cleanup() error {
	if a.cleanupCommand == nil || a.cleanupExecuted {
		return nil
	}
	
	a.log.Infof("Running ART cleanup command")
	a.cleanupExecuted = true
	
	// Use the original detonation ID for cleanup commands
	return a.executor.RunCommand(*a.cleanupCommand, a.detonationID, a.log)
}

// setupSignalHandler sets up signal handling for cleanup on interrupt
func (a *ARTDetonator) setupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		log.Info("Interrupt received, running cleanup...")
		if err := a.Cleanup(); err != nil {
			log.Warnf("Cleanup failed: %s", err.Error())
		}
		os.Exit(1)
	}()
}

// checkPrerequisites checks if prerequisites are met and optionally installs them
func (a *ARTDetonator) checkPrerequisites() error {
	if a.test == nil || len(a.test.Dependencies) == 0 {
		return nil // No prerequisites to check
	}
	
	installPrereqs := strings.ToLower(os.Getenv("THREATEST_ART_INSTALL_PREREQS")) == "true"
	var failedPrereqs []string
	
	for _, dep := range a.test.Dependencies {
		if dep.PreReqCommand == "" {
			continue // Skip dependencies without prereq check
		}
		
		// Check if prerequisite is met
		prereqCmd := a.test.interpolateCommand(dep.PreReqCommand, a.inputs)
		prereqDetonator := detonators.NewCommandDetonator(a.executor, prereqCmd)
		
		log.Debugf("Checking prerequisite: %s", dep.Description)
		if _, err := prereqDetonator.Detonate(); err != nil {
			// Prerequisite not met
			if installPrereqs && dep.GetPreReqCommand != "" {
				// Try to install prerequisite
				log.Infof("Installing prerequisite: %s", dep.Description)
				installCmd := a.test.interpolateCommand(dep.GetPreReqCommand, a.inputs)
				installDetonator := detonators.NewCommandDetonator(a.executor, installCmd)
				
				if _, err := installDetonator.Detonate(); err != nil {
					failedPrereqs = append(failedPrereqs, fmt.Sprintf("%s (installation failed: %s)", dep.Description, err.Error()))
					continue
				}
				
				// Verify prerequisite is now met
				if _, err := prereqDetonator.Detonate(); err != nil {
					failedPrereqs = append(failedPrereqs, fmt.Sprintf("%s (verification failed after installation)", dep.Description))
				} else {
					log.Infof("Successfully installed prerequisite: %s", dep.Description)
				}
			} else {
				// Cannot install or user chose not to install
				failedPrereqs = append(failedPrereqs, dep.Description)
			}
		} else {
			log.Debugf("Prerequisite satisfied: %s", dep.Description)
		}
	}
	
	if len(failedPrereqs) > 0 {
		errorMsg := fmt.Sprintf("Prerequisites not met: %s", strings.Join(failedPrereqs, ", "))
		if !installPrereqs {
			errorMsg += "\nSet THREATEST_ART_INSTALL_PREREQS=true to automatically install prerequisites."
		}
		return fmt.Errorf(errorMsg)
	}
	
	return nil
}