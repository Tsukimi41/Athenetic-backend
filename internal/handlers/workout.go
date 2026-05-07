package handlers

import (
	"net/http"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GetTodaysWorkout は本日のトレーニングメニューを返します（現状は固定モックデータ）
func GetTodaysWorkout(c echo.Context) error {
	todaysMenu := map[string]interface{}{
		"date":            time.Now().Format("2006-01-02"),
		"title":           "Upper Body Push",
		"readiness_score": 92,
		"exercises": []map[string]interface{}{
			{
				"id":          "ex-1",
				"name":        "Decline Push-up",
				"target_sets": 3,
				"history":     "前回: 12 reps (RPE 9)",
			},
			{
				"id":          "ex-2",
				"name":        "Archer Push-up",
				"target_sets": 2,
				"history":     "前回: 8 reps (RPE 9.5)",
			},
		},
	}
	return c.JSON(http.StatusOK, todaysMenu)
}

// Next.jsから送られてくるJSONの形を定義
type CompleteSetRequest struct {
	ExerciseName string  `json:"exercise_name"`
	SetNumber    int     `json:"set_number"`
	Reps         int     `json:"reps"`
	RPE          float64 `json:"rpe"`
}

// CreateWorkoutRecord は完了したセットをPostgreSQLに保存します
func CreateWorkoutRecord(c echo.Context) error {
	var req CompleteSetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "リクエストの形式が不正です"})
	}

	db := database.DB

	// 1. ダミーユーザーの確保（将来のログイン機能までの繋ぎ）
	var user models.User
	db.FirstOrCreate(&user, models.User{Email: "test@athenetic.app", Name: "Test User"})

	// 2. 今日のセッション（WorkoutSession）の確保
	var session models.WorkoutSession
	today := time.Now().Truncate(24 * time.Hour)
	db.Where("user_id = ? AND start_time >= ?", user.ID, today).
		FirstOrCreate(&session, models.WorkoutSession{
			UserID:         user.ID,
			Title:          "Daily Workout",
			StartTime:      time.Now(),
			ReadinessScore: 90,
		})

	// 3. 種目（Exercise）の確保
	var exercise models.Exercise
	db.Where("name = ?", req.ExerciseName).
		FirstOrCreate(&exercise, models.Exercise{
			Name:         req.ExerciseName,
			TargetMuscle: models.Chest, // とりあえず胸として保存
			IsBodyweight: true,
		})

	// 4. セットの記録（WorkoutSet）をデータベースに保存！
	workoutSet := models.WorkoutSet{
		SessionID:   session.ID,
		ExerciseID:  exercise.ID,
		SetNumber:   req.SetNumber,
		Reps:        req.Reps,
		RPE:         req.RPE,
		IsCompleted: true,
	}

	if err := db.Create(&workoutSet).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "データベースへの保存に失敗しました"})
	}

	// 成功レスポンス
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Record saved successfully!",
		"set_id":  workoutSet.ID,
	})
}