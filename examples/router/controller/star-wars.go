package controller

import (
	"di/example/router/controller/dto"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-path/di"
)

var swapi = "https://swapi.dev/api"

type StarWarsApi struct{}

func (s *StarWarsApi) Path() string {
	return "/star-wars"
}

func (s *StarWarsApi) Index(req *http.Request, w http.ResponseWriter) string {
	return "The Star Wars API"
}

func (s *StarWarsApi) People(req *http.Request, w http.ResponseWriter) (*dto.People, error) {
	p := &dto.People{}
	if err := s.requestSWAPI(req, "people", p); err != nil {
		return nil, err
	} else {
		slog.Info(fmt.Sprintf("People found: { Name: %s, Gender : %s }", p.Name, p.Gender))
	}
	return p, nil
}

func (s *StarWarsApi) Planet(req *http.Request, w http.ResponseWriter) (*dto.Planet, error) {
	p := &dto.Planet{}
	if err := s.requestSWAPI(req, "planets", p); err != nil {
		return nil, err
	} else {
		slog.Info(fmt.Sprintf("Planet found: { Name: %s }", p.Name))
	}
	return p, nil
}

func (s *StarWarsApi) Starship(req *http.Request, w http.ResponseWriter) (*dto.Starship, error) {
	out := &dto.Starship{}
	if err := s.requestSWAPI(req, "starships", out); err != nil {
		return nil, err
	} else {
		slog.Info(fmt.Sprintf("Starship found: { Name: %s }", out.Name))
	}
	return out, nil
}

func (s *StarWarsApi) requestSWAPI(req *http.Request, api string, out any) error {
	id := strings.TrimSpace(req.URL.Query().Get("id"))
	if id == "" {
		return errors.New("query string 'id' is required")
	}

	if res, err := http.Get(fmt.Sprintf("%s/%s/%s/", swapi, api, id)); err != nil {
		return err
	} else if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return err
	}

	return nil
}

func init() {
	di.Register(&StarWarsApi{}) // singleton
}
