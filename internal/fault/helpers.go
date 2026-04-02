package fault

import "github.com/CyberArgonaut/makakito/internal/docker"

func containerStopOptions() docker.StopOptions {
	return docker.StopOptions{}
}

func containerStartOptions() docker.StartOptions {
	return docker.StartOptions{}
}
