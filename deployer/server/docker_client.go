package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"strconv"
)

var (
	errNotFound = errors.New("not found")
)

func (s *Server) CheckIfServiceExists(ctx context.Context, username string) (bool, error) {
	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	for _, ct := range containers {
		if ct.Names[0] == "/"+username {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) createVolumeIfNotExists(ctx context.Context, port int) (string, error) {
	volumeName := fmt.Sprintf("function-%d", port)
	exists, err := s.volumeExists(ctx, volumeName)
	if err != nil {
		return "", err
	}

	if exists {
		return volumeName, nil
	}

	_, err = s.client.VolumeCreate(ctx, volume.CreateOptions{
		Name: volumeName,
	})
	if err != nil {
		return "", err
	}

	return volumeName, nil
}

func (s *Server) DeployContainer(ctx context.Context, username string) (int, error) {
	containerPort := s.nextContainerPort

	volumeName, err := s.createVolumeIfNotExists(ctx, containerPort)
	if err != nil {
		return 0, err
	}

	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: strconv.Itoa(containerPort),
	}

	portMap, err := nat.NewPort("tcp", "8081")
	if err != nil {
		return 0, err
	}
	portBinding := nat.PortMap{portMap: []nat.PortBinding{hostBinding}}

	resp, err := s.client.ContainerCreate(ctx, &container.Config{
		Image: s.image,
		ExposedPorts: nat.PortSet{
			"8081/tcp": struct{}{},
		},
		Env: []string{"FRONTEND_URL=http://localhost:3000", "FRONTEND_HOST=localhost"},
	}, &container.HostConfig{
		PortBindings: portBinding,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/data",
			},
		},
	}, nil, nil, username)
	if err != nil {
		return 0, err
	}

	if err := s.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return 0, err
	}
	fmt.Println(containerPort)

	s.nextContainerPort++
	return containerPort, nil
}

func (s *Server) DeleteServiceForUser(ctx context.Context, username string) error {
	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	for _, ct := range containers {
		if ct.Names[0] == "/"+username {
			err = s.client.ContainerStop(ctx, ct.ID, container.StopOptions{})
			if err != nil {
				return err
			}
			return s.client.ContainerRemove(ctx, ct.ID, types.ContainerRemoveOptions{})
		}
	}
	return errNotFound
}

func (s *Server) volumeExists(ctx context.Context, name string) (bool, error) {
	volumes, err := s.client.VolumeList(ctx, filters.Args{})
	if err != nil {
		return false, err
	}

	for _, volume := range volumes.Volumes {
		if volume.Name == name {
			return true, nil
		}
	}

	return false, nil
}
