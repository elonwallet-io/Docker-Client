package server

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

const (
	NAMESPACE = "default"
	TIMEOUT   = 180 * time.Second
)

func (s *Server) CheckIfServiceExists(ctx context.Context, username string) (bool, error) {
	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		if container.Names[0] == "/"+username {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) DeployContainer(ctx context.Context, username string) (int, error) {
	port := s.port
	resp, err := s.client.ContainerCreate(ctx, &container.Config{
		Image: s.image,
	}, &container.HostConfig{PortBindings: nat.PortMap{"8080": []nat.PortBinding{
		{
			HostIP:   "127.0.0.1",
			HostPort: strconv.Itoa(port),
		},
	}}}, nil, nil, username)
	if err != nil {
		return 0, err
	}
	s.port++
	if err := s.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return 0, err
	}
	fmt.Println(port)
	return port, nil
}

type NotFound struct{}

func (m *NotFound) Error() string {
	return "not found"
}

func (s *Server) DeleteServiceForUser(ctx context.Context, username string) error {
	containers, err := s.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	for _, containerr := range containers {
		if containerr.Names[0] == "/"+username {
			err = s.client.ContainerStop(ctx, containerr.ID, container.StopOptions{})
			if err != nil {
				return err
			}
			return s.client.ContainerRemove(ctx, containerr.ID, types.ContainerRemoveOptions{})
		}
	}
	return &NotFound{}
}
