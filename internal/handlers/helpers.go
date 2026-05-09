package handlers

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
)

func getUserID(c echo.Context) (uuid.UUID, error) {
	value := c.Get("user_id")
	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("missing user context")
	}
	return userID, nil
}

func parseMuscleGroup(input string) (models.TargetMuscle, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "chest":
		return models.Chest, nil
	case "back":
		return models.Back, nil
	case "legs":
		return models.Legs, nil
	case "shoulders":
		return models.Shoulders, nil
	case "core":
		return models.Core, nil
	default:
		return "", errors.New("invalid muscle group")
	}
}
