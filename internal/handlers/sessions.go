package handlers

import (
	"net/http"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/labstack/echo/v4"
)

type CreateSessionRequest struct {
	SessionDate   string `json:"session_date"`
	MuscleGroup   string `json:"muscle_group"`
	ReadinessScore int   `json:"readiness_score"`
	Title         string `json:"title"`
}

type SessionExerciseResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MuscleGroup string `json:"muscle_group"`
}

type SessionSetResponse struct {
	ID            string                  `json:"id"`
	Exercise      SessionExerciseResponse `json:"exercise"`
	WeightKg      float64                 `json:"weight_kg"`
	RepsCompleted int                     `json:"reps_completed"`
	TargetReps    int                     `json:"target_reps"`
	RPE           float64                 `json:"rpe"`
	RIR           int                     `json:"rir"`
	TargetRepsNext int                    `json:"target_reps_next"`
	CreatedAt     time.Time               `json:"created_at"`
}

type SessionResponse struct {
	SessionID      string               `json:"session_id"`
	SessionDate    string               `json:"session_date"`
	MuscleGroup    string               `json:"muscle_group"`
	ReadinessScore int                  `json:"readiness_score"`
	Sets           []SessionSetResponse `json:"sets"`
}

func CreateSession(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req CreateSessionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.SessionDate == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "session_date is required"})
	}

	sessionDate, err := time.Parse("2006-01-02", req.SessionDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid session_date"})
	}

	muscleGroup, err := parseMuscleGroup(req.MuscleGroup)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid muscle_group"})
	}

	title := req.Title
	if title == "" {
		title = defaultTitleForMuscle(muscleGroup)
	}

	session := models.WorkoutSession{
		UserID:         userID,
		Title:          title,
		SessionDate:    sessionDate,
		MuscleGroup:    muscleGroup,
		StartTime:      time.Now(),
		ReadinessScore: req.ReadinessScore,
	}

	db := database.DB
	var existing models.WorkoutSession
	if err := db.Where("user_id = ? AND DATE(session_date) = ? AND LOWER(muscle_group) = LOWER(?)", userID, sessionDate.Format("2006-01-02"), muscleGroup).First(&existing).Error; err == nil {
		return c.JSON(http.StatusOK, map[string]string{"session_id": existing.ID.String()})
	}

	if err := db.Create(&session).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"session_id": session.ID.String()})
}

func GetSessionsByDate(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	dateStr := c.Param("date")
	sessionDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid date"})
	}

	muscleGroupParam := c.QueryParam("muscle_group")

	db := database.DB
	query := db.Where("user_id = ? AND DATE(session_date) = ?", userID, sessionDate.Format("2006-01-02"))
	if muscleGroupParam != "" {
		query = query.Where("LOWER(muscle_group) = LOWER(?)", muscleGroupParam)
	}

	var sessions []models.WorkoutSession
	if err := query.Find(&sessions).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load sessions"})
	}

	responses := make([]SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		var sets []models.WorkoutSet
		if err := db.Where("session_id = ?", session.ID).Order("created_at asc").Find(&sets).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load sets"})
		}

		exerciseIDs := make([]string, 0, len(sets))
		for _, set := range sets {
			exerciseIDs = append(exerciseIDs, set.ExerciseID.String())
		}

		exerciseMap := map[string]models.Exercise{}
		if len(exerciseIDs) > 0 {
			var exercises []models.Exercise
			if err := db.Where("id IN ?", exerciseIDs).Find(&exercises).Error; err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load exercises"})
			}
			for _, ex := range exercises {
				exerciseMap[ex.ID.String()] = ex
			}
		}

		setResponses := make([]SessionSetResponse, 0, len(sets))
		for _, set := range sets {
			ex := exerciseMap[set.ExerciseID.String()]
			setResponses = append(setResponses, SessionSetResponse{
				ID: set.ID.String(),
				Exercise: SessionExerciseResponse{
					ID:          ex.ID.String(),
					Name:        ex.Name,
					MuscleGroup: string(ex.TargetMuscle),
				},
				WeightKg:      set.Weight,
				RepsCompleted: set.Reps,
				TargetReps:    set.TargetReps,
				RPE:           set.RPE,
				RIR:           set.RIR,
				TargetRepsNext: calculateNextReps(set.RPE, set.Reps, set.TargetReps),
				CreatedAt:     set.CreatedAt,
			})
		}

		responses = append(responses, SessionResponse{
			SessionID:      session.ID.String(),
			SessionDate:    session.SessionDate.Format("2006-01-02"),
			MuscleGroup:    string(session.MuscleGroup),
			ReadinessScore: session.ReadinessScore,
			Sets:           setResponses,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"session_date": sessionDate.Format("2006-01-02"),
		"sessions":     responses,
	})
}

func defaultTitleForMuscle(muscleGroup models.TargetMuscle) string {
	switch muscleGroup {
	case models.Chest:
		return "Upper Body Push"
	case models.Back:
		return "Upper Body Pull"
	case models.Legs:
		return "Lower Body Strength"
	case models.Shoulders:
		return "Shoulder Development"
	case models.Core:
		return "Core Stability"
	default:
		return "Training Session"
	}
}
