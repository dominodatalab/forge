package v1alpha1

// BuildState represents a phase in the build process.
type BuildState string

const (
	// Initialized indicates that a new build has been intercepted by the controller.
	Initialized BuildState = "Initialized"

	// Building indicates that a build that is currently running.
	Building BuildState = "Building"

	// Completed indicates that a build has finished successfully.
	Completed BuildState = "Completed"

	// Failed indicates that a build encountered an error during the build process.
	Failed BuildState = "Failed"
)
