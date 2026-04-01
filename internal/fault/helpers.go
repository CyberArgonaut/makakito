package fault

import "github.com/docker/docker/api/types/container"

func containerStopOptions() container.StopOptions {
	return container.StopOptions{}
}

func containerStartOptions() container.StartOptions {
	return container.StartOptions{}
}
