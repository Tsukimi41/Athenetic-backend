package handlers

import (
	"net/http"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CreateSetRequest struct {
	SessionID     string  `json:"session_id"`
	ExerciseID    string  `json:"exercise_id"`
	WeightKg      float64 `json:"weight_kg"`
	RepsCompleted int     `json:"reps_completed"`
	RPE           float64 `json:"rpe"`
	RIR           int     `json:"rir"`
	TargetReps    int     `json:"target_reps"`
}

func CreateSet(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req CreateSetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid session_id"})
	}

	exerciseID, err := uuid.Parse(req.ExerciseID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid exercise_id"})
	}

	if req.RepsCompleted <= 0 || req.RepsCompleted > 200 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "reps_completed out of range"})
	}
	if req.RPE < 0 || req.RPE > 10 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "rpe must be 0-10"})
	}
	if req.RIR < 0 || req.RIR > 10 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "rir must be 0-10"})
	}
	if req.WeightKg < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "weight_kg must be positive"})
	}
	if req.TargetReps <= 0 {
		req.TargetReps = req.RepsCompleted
	}

	db := database.DB

	var session models.WorkoutSession
	if err := db.First(&session, "id = ? AND user_id = ?", sessionID, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	var exercise models.Exercise
	if err := db.First(&exercise, "id = ?", exerciseID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "exercise not found"})
	}

	var existingCount int64
	_ = db.Model(&models.WorkoutSet{}).Where("session_id = ? AND exercise_id = ?", sessionID, exerciseID).Count(&existingCount)
	setNumber := int(existingCount) + 1

	set := models.WorkoutSet{
		SessionID:   sessionID,
		ExerciseID:  exerciseID,
		SetNumber:   setNumber,
		Reps:        req.RepsCompleted,
		TargetReps:  req.TargetReps,
		Weight:      req.WeightKg,
		RPE:         req.RPE,
		RIR:         req.RIR,
		IsCompleted: true,
		CreatedAt:   time.Now(),
	}

	if err := db.Create(&set).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save set"})
	}

	nextReps := calculateNextReps(req.RPE, req.RepsCompleted, req.TargetReps)

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"set_id":           set.ID.String(),
		"target_reps_next": nextReps,
		"note":             "Set logged successfully",
	})
}

func GetPreviousSet(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	exerciseID, err := uuid.Parse(c.Param("exercise_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid exercise_id"})
	}

	db := database.DB

	var set models.WorkoutSet
	result := db.Model(&models.WorkoutSet{}).
		Joins("JOIN workout_sessions ws ON ws.id = workout_sets.session_id").
		Where("ws.user_id = ?", userID).
		Where("workout_sets.exercise_id = ?", exerciseID).
		Order("workout_sets.created_at desc").
		First(&set)

	if result.Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no previous set found"})
	}

	var session models.WorkoutSession
	_ = db.First(&session, "id = ?", set.SessionID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"weight_kg":        set.Weight,
		"reps_completed":   set.Reps,
		"rpe":              set.RPE,
		"target_reps_next": calculateNextReps(set.RPE, set.Reps, set.TargetReps),
		"session_date":     session.SessionDate.Format("2006-01-02"),
		"notes":            "Last session data",
	})
}
