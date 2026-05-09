package handlers

import (
	"net/http"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/labstack/echo/v4"
)

func ListExercises(c echo.Context) error {
	muscleGroup := c.QueryParam("muscle_group")

	db := database.DB
	query := db.Model(&models.Exercise{})
	if muscleGroup != "" {
		if _, err := parseMuscleGroup(muscleGroup); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid muscle_group"})
		}
		query = query.Where("LOWER(target_muscle) = LOWER(?)", muscleGroup)
	}

	var exercises []models.Exercise
	if err := query.Order("name asc").Find(&exercises).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load exercises"})
	}

	response := make([]map[string]interface{}, 0, len(exercises))
	for _, ex := range exercises {
		response = append(response, map[string]interface{}{
			"id":                  ex.ID,
			"name":                ex.Name,
			"muscle_group":        string(ex.TargetMuscle),
			"default_target_sets": ex.DefaultTargetSets,
			"is_bodyweight":       ex.IsBodyweight,
		})
	}

	return c.JSON(http.StatusOK, response)
}
