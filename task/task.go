package task

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"io"
	"log"
	"math"
	"os"
	"time"
)

/*
  - A note about Containers Network Mode:
    default value for NetworkMode: "bridge", we set it to "host" when calling HTTP POST request;
    so we can use the tim's echo image which was used for chapter 9.
    warning: if you're using Windows for development,
    go to DockerDesktop's setting -> Resources -> Network -> Enable host networking

  - We also could remove HostPorts attribute since we are using 'host' mode for NetworkMode;
    but to keep things in consistent with the book, we keep it.
*/
type Task struct {
	ID            uuid.UUID
	ContainerID   string
	Name          string
	State         State
	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	ExposedPort   nat.PortSet
	PortBindings  nat.PortMap
	NetworkMode   container.NetworkMode
	RestartPolicy string
	StartTime     time.Time
	FinishTime    time.Time
	HostPorts     nat.PortMap
	HealthCheck   string
	RestartCount  int
}

type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	ExposedPort   nat.PortSet
	PortBindings  nat.PortMap
	NetworkMode   container.NetworkMode
	Cmd           []string
	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
}

func NewConfig(t *Task) *Config {
	return &Config{
		Name:          t.Name,
		ExposedPort:   t.ExposedPort,
		PortBindings:  t.PortBindings,
		Image:         t.Image,
		Cpu:           t.Cpu,
		Memory:        t.Memory,
		Disk:          t.Disk,
		RestartPolicy: t.RestartPolicy,
		NetworkMode:   t.NetworkMode,
	}
}

type Docker struct {
	Client *client.Client
	Config Config
}

func NewDocker(c *Config) *Docker {
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	return &Docker{
		Client: dc,
		Config: *c,
	}
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerId string
	Result      string
}

type DockerInspectResponse struct {
	Error     error
	Container *types.ContainerJSON
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()

	reader, err := d.Client.ImagePull(
		ctx,
		d.Config.Image,
		types.ImagePullOptions{},
	)
	if err != nil {
		log.Printf("Error pulling image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy{
		Name: d.Config.RestartPolicy,
	}

	r := container.Resources{
		Memory:   d.Config.Memory,
		NanoCPUs: int64(d.Config.Cpu * math.Pow(10, 9)),
	}

	cc := container.Config{
		Image:        d.Config.Image,
		Env:          d.Config.Env,
		ExposedPorts: d.Config.ExposedPort,
		Tty:          false,
	}

	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true,
		PortBindings:    d.Config.PortBindings,
		NetworkMode:     d.Config.NetworkMode,
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		log.Printf("Error creating container using image: %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	if err = d.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("Error starting container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	out, err := d.Client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		log.Printf("Error getting logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return DockerResult{
		ContainerId: resp.ID,
		Action:      "start",
		Result:      "success",
	}
}

func (d *Docker) Stop(id string) DockerResult {
	log.Printf("Attempting to stop container %v\n", id)
	ctx := context.Background()
	if err := d.Client.ContainerStop(ctx, id, nil); err != nil {
		log.Printf("Error stopping container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	err := d.Client.ContainerRemove(
		ctx,
		id,
		types.ContainerRemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false,
			Force:         false,
		})
	if err != nil {
		log.Printf("Error removing container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	return DockerResult{Action: "stop", Result: "success", Error: nil}
}

func (d *Docker) Inspect(containerId string) DockerInspectResponse {
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	ctx := context.Background()
	resp, err := dc.ContainerInspect(ctx, containerId)
	if err != nil {
		log.Printf("Error inspecting container %s: %v\n", containerId, err)
		return DockerInspectResponse{Error: err}
	}

	return DockerInspectResponse{Container: &resp}
}
