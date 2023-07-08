package server

import (
	"deployer/server/docker"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (s *Server) HandleDeployContainer() echo.HandlerFunc {
	type input struct {
		Name string `json:"name"`
	}
	type output struct {
		URL string `json:"url"`
	}
	return func(c echo.Context) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}

		url, err := s.client.DeployContainer(in.Name, c.Request().Context())
		if err == docker.ErrAlreadyExists {
			log.Debug().Caller().Str("name", in.Name).Msg("container does already exist")
			return echo.NewHTTPError(http.StatusBadRequest, "A container with this name does already exist").SetInternal(err)
		}
		if err != nil {
			log.Error().Err(err).Caller().Str("name", in.Name).Msg("failed to deploy container")
			return err
		}

		waitForContainerToStart(url)

		log.Debug().Caller().Str("name", in.Name).Str("url", url).Msg("container deployed successfully")
		return c.JSON(http.StatusCreated, output{url})
	}
}

func (s *Server) HandleRemoveContainer() echo.HandlerFunc {
	type input struct {
		Name string `param:"id"`
	}
	return func(c echo.Context) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}

		err := s.client.RemoveContainerAndVolume(in.Name, c.Request().Context())
		if err == docker.ErrNotFound {
			log.Debug().Caller().Str("name", in.Name).Msg("container does not exist")
			return echo.NewHTTPError(http.StatusNotFound, "A container with this name does not exist").SetInternal(err)
		}
		if err != nil {
			log.Error().Err(err).Caller().Str("name", in.Name).Msg("failed to remove container")
			return err
		}

		log.Debug().Caller().Str("name", in.Name).Msg("container removed successfully")
		return nil
	}
}
