package server

import (
	"context"
	"deployer/server/docker"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type Server struct {
	echo   *echo.Echo
	client *docker.Client
}

func New() (*Server, error) {
	e := echo.New()
	e.Server.ReadTimeout = 5 * time.Second
	e.Server.WriteTimeout = 30 * time.Second
	e.Server.IdleTimeout = 120 * time.Second

	c, err := docker.NewClient()
	if err != nil {
		return nil, err
	}

	return &Server{
		echo:   e,
		client: c,
	}, nil
}

func (s *Server) Run() (err error) {
	s.echo.POST("/enclaves", s.HandleDeployContainer())
	s.echo.DELETE("/enclaves/:id", s.HandleRemoveContainer())

	err = s.echo.Start(":8082")
	if err == http.ErrServerClosed {
		err = nil
	}
	return
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.echo.Shutdown(ctx)
}
