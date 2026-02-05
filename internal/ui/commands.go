package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"tinyd/internal/docker"
	"tinyd/internal/types"
)

// tickCmd creates a periodic tick for auto-refresh
func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return types.TickMsg(t)
	})
}

// animationTickCmd creates a fast tick for status animations
func animationTickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return types.AnimationTickMsg(t)
	})
}

// fetchContainersCmd fetches containers from Docker
func (m *Model) fetchContainersCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		containers, err := m.docker.FetchContainers(ctx)
		if err != nil {
			return types.ErrMsg(err)
		}
		return types.ContainerListMsg(containers)
	}
}

// fetchImagesCmd fetches images from Docker
func (m *Model) fetchImagesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		images, err := m.docker.FetchImages(ctx)
		if err != nil {
			return types.ErrMsg(err)
		}
		return types.ImageListMsg(images)
	}
}

// fetchVolumesCmd fetches volumes from Docker
func (m *Model) fetchVolumesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		volumes, err := m.docker.FetchVolumes(ctx)
		if err != nil {
			return types.ErrMsg(err)
		}
		return types.VolumeListMsg(volumes)
	}
}

// fetchNetworksCmd fetches networks from Docker
func (m *Model) fetchNetworksCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		networks, err := m.docker.FetchNetworks(ctx)
		if err != nil {
			return types.ErrMsg(err)
		}
		return types.NetworkListMsg(networks)
	}
}

// startContainerCmd starts a container
func (m *Model) startContainerCmd(containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		if err := m.docker.StartContainer(ctx, containerID); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Container " + containerName + " started")
	}
}

// stopContainerCmd stops a container
func (m *Model) stopContainerCmd(containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		if err := m.docker.StopContainer(ctx, containerID); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Container " + containerName + " stopped")
	}
}

// restartContainerCmd restarts a container
func (m *Model) restartContainerCmd(containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		if err := m.docker.RestartContainer(ctx, containerID); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Container " + containerName + " restarted")
	}
}

// deleteContainerCmd deletes a container
func (m *Model) deleteContainerCmd(containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		if err := m.docker.DeleteContainer(ctx, containerID, true); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Container " + containerName + " deleted")
	}
}

// getContainerLogsCmd retrieves container logs
func (m *Model) getContainerLogsCmd(containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		logs, err := m.docker.GetContainerLogs(ctx, containerID, "100")
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.LogsMsg(logs)
	}
}

// inspectContainerCmd retrieves container inspect data
func (m *Model) inspectContainerCmd(containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		inspect, err := m.docker.InspectContainer(ctx, containerID)
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.InspectMsg(inspect)
	}
}

// deleteImageCmd deletes an image
func (m *Model) deleteImageCmd(imageID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		if err := m.docker.DeleteImage(ctx, imageID, false); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Image deleted successfully")
	}
}

// pullImageCmd pulls an image
func (m *Model) pullImageCmd(imageName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithCustomTimeout(docker.TimeoutLong)
		defer cancel()

		if err := m.docker.PullImage(ctx, imageName); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Image " + imageName + " pulled successfully")
	}
}

// inspectImageCmd retrieves image inspect data
func (m *Model) inspectImageCmd(imageID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		inspect, err := m.docker.InspectImage(ctx, imageID)
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.InspectMsg(inspect)
	}
}

// runContainerCmd creates and runs a container from an image
func (m *Model) runContainerCmd() tea.Cmd {
	return func() tea.Msg {
		if m.selectedImage == nil {
			return types.ActionErrorMsg("No image selected")
		}

		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		containerID, err := m.docker.RunContainer(ctx, m.selectedImage, m.runContainerName, m.runPorts, m.runVolumes, m.runEnvVars)
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Container started: " + containerID)
	}
}

// deleteVolumeCmd deletes a volume
func (m *Model) deleteVolumeCmd(volumeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		if err := m.docker.DeleteVolume(ctx, volumeName, true); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Volume " + volumeName + " deleted")
	}
}

// inspectVolumeCmd retrieves volume inspect data
func (m *Model) inspectVolumeCmd(volumeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		inspect, err := m.docker.InspectVolume(ctx, volumeName)
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.InspectMsg(inspect)
	}
}

// deleteNetworkCmd deletes a network
func (m *Model) deleteNetworkCmd(networkID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		if err := m.docker.DeleteNetwork(ctx, networkID); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Network deleted successfully")
	}
}

// inspectNetworkCmd retrieves network inspect data
func (m *Model) inspectNetworkCmd(networkID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithTimeout()
		defer cancel()

		inspect, err := m.docker.InspectNetwork(ctx, networkID)
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.InspectMsg(inspect)
	}
}
