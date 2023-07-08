package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"strconv"
	"sync"
)

const (
	firstContainerPort = 8083
	containerImageName = "function"
	frontendURL        = "http://localhost:3000"
	frontendHost       = "localhost"
	backendURL         = "http://host.docker.internal:8080"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource does already exist")
)

type Client struct {
	client            *client.Client
	mu                sync.Mutex
	nextContainerPort int
}

func NewClient() (*Client, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker daemon: %w", err)
	}

	return &Client{
		client:            c,
		mu:                sync.Mutex{},
		nextContainerPort: firstContainerPort,
	}, nil
}

func (c *Client) DeployContainer(name string, ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	exists, err := c.serviceExists(name, ctx)
	if err != nil {
		return "", err
	}
	if exists {
		return "", ErrAlreadyExists
	}

	err = c.createVolumeIfNotExists(name, ctx)
	if err != nil {
		return "", err
	}

	containerPort := c.nextContainerPort

	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: strconv.Itoa(containerPort),
	}

	portMap, err := nat.NewPort("tcp", "8081")
	if err != nil {
		return "", fmt.Errorf("failed to create portmap: %w", err)
	}
	portBinding := nat.PortMap{
		portMap: []nat.PortBinding{hostBinding},
	}

	resp, err := c.client.ContainerCreate(ctx, &container.Config{
		Image: containerImageName,
		ExposedPorts: nat.PortSet{
			"8081/tcp": struct{}{},
		},
		Env: []string{
			fmt.Sprintf("FRONTEND_URL=%s", frontendURL),
			fmt.Sprintf("FRONTEND_HOST=%s", frontendHost),
			fmt.Sprintf("BACKEND_URL=%s", backendURL),
			"USE_INSECURE_HTTP=true",
		},
	}, &container.HostConfig{
		PortBindings: portBinding,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: name,
				Target: "/data",
			},
		},
	}, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	err = c.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	c.nextContainerPort++

	return fmt.Sprintf("http://host.docker.internal:%d", containerPort), nil
}

func (c *Client) RemoveContainerAndVolume(name string, ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ct, err := c.findContainer(name, ctx)
	if err != nil {
		return err
	}

	err = c.removeContainer(ct, ctx)
	if err != nil {
		return err
	}

	err = c.removeVolume(name, ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) removeContainer(ct *types.Container, ctx context.Context) error {
	err := c.client.ContainerStop(ctx, ct.ID, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	err = c.client.ContainerRemove(ctx, ct.ID, types.ContainerRemoveOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

func (c *Client) removeVolume(name string, ctx context.Context) error {
	err := c.client.VolumeRemove(ctx, name, true)
	if err != nil {
		return fmt.Errorf("failed to remove volume: %w", err)
	}

	return nil
}

func (c *Client) findContainer(name string, ctx context.Context) (*types.Container, error) {
	containers, err := c.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing containers: %w", err)
	}

	containerName := fmt.Sprintf("/%s", name)
	for _, ct := range containers {
		if ct.Names[0] == containerName {
			return &ct, nil
		}
	}

	return nil, ErrNotFound
}

func (c *Client) findVolume(name string, ctx context.Context) (*volume.Volume, error) {
	volumes, err := c.client.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing volumes: %w", err)
	}

	for _, v := range volumes.Volumes {
		if v.Name == name {
			return v, nil
		}
	}

	return nil, ErrNotFound
}

func (c *Client) createVolumeIfNotExists(name string, ctx context.Context) error {
	exists, err := c.volumeExists(name, ctx)
	if err != nil {
		return err
	}

	if !exists {
		_, err := c.client.VolumeCreate(ctx, volume.CreateOptions{
			Name: name,
		})

		if err != nil {
			return fmt.Errorf("failed to create volume: %w", err)
		}
	}

	return nil
}

func (c *Client) serviceExists(name string, ctx context.Context) (bool, error) {
	_, err := c.findContainer(name, ctx)
	if err == ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *Client) volumeExists(name string, ctx context.Context) (bool, error) {
	_, err := c.findVolume(name, ctx)
	if err == ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
