package ui

import (
	"context"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"tinyd/internal/dmr"
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

// stopContainerCmd stops a container. Uses a longer timeout than the default
// 10s because Docker's own SIGTERM grace period is 10s — anything shorter
// would race with Docker's own clock and surface a spurious DeadlineExceeded
// even when the container actually stopped.
func (m *Model) stopContainerCmd(containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithCustomTimeout(30 * time.Second)
		defer cancel()

		if err := m.docker.StopContainer(ctx, containerID); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Container " + containerName + " stopped")
	}
}

// restartContainerCmd restarts a container. Same rationale as stopContainerCmd
// — restart is stop+start with a 10s grace, so we give ourselves headroom.
func (m *Model) restartContainerCmd(containerID, containerName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithCustomTimeout(30 * time.Second)
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

// searchImagesCmd searches Docker Hub for images
func (m *Model) searchImagesCmd(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithCustomTimeout(docker.TimeoutMedium)
		defer cancel()

		results, err := m.docker.SearchImages(ctx, query, 25)
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ImageSearchMsg(results)
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

// updateImageCmd re-pulls an existing image to update it to the latest
// version under its tag. Same call as pullImageCmd, different status text.
func (m *Model) updateImageCmd(imageName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.docker.WithCustomTimeout(docker.TimeoutLong)
		defer cancel()

		if err := m.docker.PullImage(ctx, imageName); err != nil {
			return types.ActionErrorMsg("Failed to update " + imageName + ": " + err.Error())
		}
		return types.ActionSuccessMsg("Updated " + imageName + " to latest")
	}
}

// pullSearchCompleteCmd pulls an image and ensures the user is returned to the
// list view (the generic pull command returns ActionSuccessMsg which doesn't
// reset currentView). We chain a final state-reset message so completion exits
// the pull flow cleanly.
func (m *Model) pullSearchCompleteCmd(imageName string) tea.Cmd {
	// Use the longer pull timeout
	return func() tea.Msg {
		ctx, cancel := m.docker.WithCustomTimeout(docker.TimeoutLong)
		defer cancel()

		if err := m.docker.PullImage(ctx, imageName); err != nil {
			return types.ActionErrorMsg("Failed to pull " + imageName + ": " + err.Error())
		}
		return types.ActionSuccessMsg("Pulled " + imageName)
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

// --- Docker Model Runner commands ---

// probeDMRCmd checks whether DMR is reachable. Runs once at startup; the
// Models tab uses the result to decide between rendering models and a
// "DMR not enabled" empty state.
func (m *Model) probeDMRCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.dmr.WithCustomTimeout(2 * time.Second)
		defer cancel()
		return types.DMRAvailableMsg(m.dmr.Available(ctx))
	}
}

// fetchModelsCmd lists local DMR models. Returns ErrMsg silently when DMR
// is unavailable so a missing runner doesn't spam the action bar.
func (m *Model) fetchModelsCmd() tea.Cmd {
	return func() tea.Msg {
		if !m.dmrAvailable {
			return types.ModelListMsg{}
		}
		ctx, cancel := m.dmr.WithTimeout()
		defer cancel()

		models, err := m.dmr.FetchModels(ctx)
		if err != nil {
			return types.ErrMsg(err)
		}
		return types.ModelListMsg(models)
	}
}

// inspectModelCmd retrieves model inspect data
func (m *Model) inspectModelCmd(ref string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.dmr.WithTimeout()
		defer cancel()
		out, err := m.dmr.InspectModel(ctx, ref)
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.InspectMsg(out)
	}
}

// deleteModelCmd removes a local model
func (m *Model) deleteModelCmd(ref string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.dmr.WithCustomTimeout(dmr.TimeoutMedium)
		defer cancel()
		if err := m.dmr.DeleteModel(ctx, ref); err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ActionSuccessMsg("Model " + ref + " deleted")
	}
}

// searchModelsCmd searches Docker Hub's ai/ namespace
func (m *Model) searchModelsCmd(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.dmr.WithCustomTimeout(dmr.TimeoutMedium)
		defer cancel()
		results, err := m.dmr.SearchModels(ctx, query, 25)
		if err != nil {
			return types.ActionErrorMsg(err.Error())
		}
		return types.ModelSearchMsg(results)
	}
}

// pullModelCmd pulls a DMR model
func (m *Model) pullModelCmd(ref string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.dmr.WithCustomTimeout(dmr.TimeoutLong)
		defer cancel()
		if err := m.dmr.PullModel(ctx, ref); err != nil {
			return types.ActionErrorMsg("Failed to pull " + ref + ": " + err.Error())
		}
		return types.ActionSuccessMsg("Pulled " + ref)
	}
}

// runModelCmd opens an interactive chat REPL via `docker model run <ref>`.
// Suspends the TUI for the duration of the chat (same pattern as exec).
//
// Wraps the command in a shell so a failure pauses for an ENTER keypress
// before tinyd reclaims the alt-screen — otherwise the error message
// flashes for a fraction of a second and the user just sees a blink.
// Strips the `docker.io/` prefix because the CLI expects the short form.
func (m *Model) runModelCmd(ref string) tea.Cmd {
	if _, err := exec.LookPath("docker"); err != nil {
		return func() tea.Msg {
			return types.ActionErrorMsg("docker binary not found in PATH")
		}
	}
	ref = strings.TrimPrefix(ref, "docker.io/")

	// Quote-escape the ref so it's safe inside the single-quoted shell
	// argument below. Single quotes in refs are vanishingly rare but the
	// escape is cheap insurance.
	safeRef := strings.ReplaceAll(ref, "'", `'"'"'`)
	script := "docker model run '" + safeRef + "'; ec=$?; " +
		"if [ $ec -ne 0 ]; then echo; echo \"[docker model run exited with code $ec — press ENTER to return to tinyd]\"; read _; fi"

	c := exec.Command("sh", "-c", script)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return types.ActionErrorMsg("docker model run " + ref + ": " + err.Error())
		}
		return types.ActionSuccessMsg("Exited model REPL")
	})
}

// execContainerCmd opens an interactive shell in the container
func (m *Model) execContainerCmd(containerID string) tea.Cmd {
	// Try /bin/bash first
	c := exec.Command("docker", "exec", "-it", containerID, "/bin/bash")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			// Fallback to /bin/sh if bash doesn't exist
			c := exec.Command("docker", "exec", "-it", containerID, "/bin/sh")
			return tea.ExecProcess(c, func(err error) tea.Msg {
				if err != nil {
					return types.ActionErrorMsg("Failed to exec: " + err.Error())
				}
				return nil
			})()
		}
		return nil
	})
}

// startChatCmd opens a streaming chat request to DMR and returns a
// ChatStartedMsg carrying the SSE reader. Subsequent token reads are
// scheduled by the Update handler via readChatChunkCmd.
//
// We deliberately don't cancel the ctx when the request returns — the
// reader is still active and the user closes it (via ESC or end-of-stream)
// through closeChatStream on the model. The ctx is bounded by a 10-minute
// hard ceiling so a stuck stream eventually unwinds.
func (m *Model) startChatCmd(ref string, messages []types.ChatMessage) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		res, err := m.dmr.ChatStream(ctx, ref, messages)
		if err != nil {
			cancel()
			return types.ChatTokenMsg{Done: true, Err: err.Error()}
		}
		// Tie the ctx cancel to the response body close so the stream
		// shuts down cleanly when the UI calls closeChatStream.
		res.Body = &cancelOnClose{Closer: res.Body, cancel: cancel}
		return types.ChatStartedMsg{Reader: res.Reader, Body: res.Body}
	}
}

// cancelOnClose pairs an io.Closer with a context cancel, so closing the
// body also cancels the request context.
type cancelOnClose struct {
	Closer interface{ Close() error }
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.Closer.Close()
}

// readChatChunkCmd reads one SSE chunk from the active chat reader. It is
// re-scheduled by the Update handler after each non-terminal token until
// Done becomes true.
func (m *Model) readChatChunkCmd() tea.Cmd {
	reader := m.chatReader
	return func() tea.Msg {
		if reader == nil {
			return types.ChatTokenMsg{Done: true, Err: "chat reader is nil"}
		}
		token, done, err := dmr.NextChatToken(reader)
		if err != nil {
			return types.ChatTokenMsg{Done: true, Err: err.Error()}
		}
		return types.ChatTokenMsg{Token: token, Done: done}
	}
}

// fetchModelTagsCmd retrieves the tag list for a Hub model repo. Sent to
// the Update loop as a ModelTagsMsg.
func (m *Model) fetchModelTagsCmd(repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := m.dmr.WithCustomTimeout(dmr.TimeoutMedium)
		defer cancel()
		tags, err := m.dmr.FetchModelTags(ctx, repo)
		if err != nil {
			return types.ActionErrorMsg("Failed to load tags: " + err.Error())
		}
		return types.ModelTagsMsg{Repo: repo, Tags: tags}
	}
}

// copyToClipboardCmd pipes text to the OS clipboard. Uses pbcopy (macOS),
// wl-copy / xclip / xsel (Linux), or clip.exe (Windows / WSL). Surfaces a
// clear status message either way.
func (m *Model) copyToClipboardCmd(text, label string) tea.Cmd {
	return func() tea.Msg {
		candidates := [][]string{
			{"pbcopy"},
			{"wl-copy"},
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
			{"clip.exe"},
		}
		for _, args := range candidates {
			if _, err := exec.LookPath(args[0]); err != nil {
				continue
			}
			cmd := exec.Command(args[0], args[1:]...)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				continue
			}
			if err := cmd.Start(); err != nil {
				continue
			}
			_, _ = stdin.Write([]byte(text))
			stdin.Close()
			if err := cmd.Wait(); err == nil {
				return types.ActionSuccessMsg(label + " copied to clipboard")
			}
		}
		return types.ActionErrorMsg("no clipboard tool found (install pbcopy, wl-copy, xclip, xsel, or clip.exe)")
	}
}
