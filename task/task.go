package task

import "github.com/stephenrlouie/travis/model"

// Tasker exposes the ability to run tasks
type Tasker interface {
	// Run a ServiceOperation
	// Returns an error if ServiceOperation is unable to be started
	Run(*model.ServiceOperation) error

	// Stop a running ServiceOperation
	// Returns an error if ServiceOperation is unable to be stopped
	Remove(*model.ServiceOperation) error

	Logs(*model.ServiceOperation) (string, error)

	Status(*model.ServiceOperation) (string, error)

	Progress(*model.ServiceOperation) (string, error)

	Outputs(*model.ServiceOperation) (map[string][]string, error)
}
