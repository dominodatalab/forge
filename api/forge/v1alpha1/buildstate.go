package v1alpha1

// BuildState represents a phase in the build process.
type BuildState string

const (
	// BuildStateInitialized indicates that a new build has been intercepted by the controller.
	BuildStateInitialized BuildState = "Initialized"

	// BuildStateBuilding indicates that a build that is currently running.
	BuildStateBuilding BuildState = "Building"

	// BuildStateCompleted indicates that a build has finished successfully.
	BuildStateCompleted BuildState = "Completed"

	// BuildStateFailed indicates that a build encountered an error during the build process.
	BuildStateFailed BuildState = "Failed"
)
