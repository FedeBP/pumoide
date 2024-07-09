package services

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type EnvironmentService struct {
	DefaultPath string
	Logger      *logrus.Logger
}

func NewEnvironmentService(defaultPath string, logger *logrus.Logger) *EnvironmentService {
	return &EnvironmentService{
		DefaultPath: defaultPath,
		Logger:      logger,
	}
}

func (s *EnvironmentService) GetEnvironments() ([]domain.Environment, error) {
	files, err := filepath.Glob(filepath.Join(s.DefaultPath, "*.json"))
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToReadEnvironment, err)
	}

	var environments []domain.Environment
	for _, file := range files {
		environment, err := domain.LoadEnvironment(s.DefaultPath, filepath.Base(file[:len(file)-5]))
		if err != nil {
			s.Logger.Printf(constants.ErrFailedToLoadEnvironment+" %s: %v", file, err)
			continue
		}
		environments = append(environments, *environment)
	}

	return environments, nil
}

func (s *EnvironmentService) CreateEnvironment(environment *domain.Environment) error {
	environment.ID = uuid.New().String()
	return environment.Save(s.DefaultPath)
}

func (s *EnvironmentService) UpdateEnvironment(id string, updatedEnvironment *domain.Environment) error {
	existingEnvironment, err := domain.LoadEnvironment(s.DefaultPath, id)
	if err != nil {
		return errors.NewAppError(http.StatusNotFound, constants.ErrEnvironmentNotFound, err)
	}

	existingEnvironment.Name = updatedEnvironment.Name
	existingEnvironment.Variables = updatedEnvironment.Variables

	return existingEnvironment.Save(s.DefaultPath)
}

func (s *EnvironmentService) DeleteEnvironment(id string) error {
	filePath := filepath.Join(s.DefaultPath, id+".json")
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NewAppError(http.StatusNotFound, constants.ErrEnvironmentNotFound, err)
		}
		return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToDeleteEnvironment, err)
	}
	return nil
}
