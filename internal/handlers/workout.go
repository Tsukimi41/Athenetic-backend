// internal/handlers/workout.go
package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GetTodaysWorkout は本日のトレーニングメニューを返します
func GetTodaysWorkout(c echo.Context) error {
	// ※本来はここでDBにアクセスし、ユーザーの過去の記録や
	// その日の曜日、レディネススコアに基づいてメニューを動的に生成します。

	// モックデータ: 月曜日の「Upper Body Push」メニュー
	todaysMenu := map[string]interface{}{
		"date":            time.Now().Format("2006-01-02"),
		"title":           "Upper Body Push",
		"readiness_score": 92,
		"exercises": []map[string]interface{}{
			{
				"id":          uuid.New().String(),
				"name":        "Decline Push-up",
				"target_sets": 3,
				"history":     "前回: 12 reps (RPE 9)", // 漸進性過負荷の指標
			},
			{
				"id":          uuid.New().String(),
				"name":        "Archer Push-up",
				"target_sets": 2,
				"history":     "前回: 8 reps (RPE 9.5)",
			},
			{
				"id":          uuid.New().String(),
				"name":        "Narrow Push-up",
				"target_sets": 2,
				"history":     "前回: 10 reps (RPE 10)",
			},
		},
	}

	// JSONとしてクライアント（Next.js）に返却
	return c.JSON(http.StatusOK, todaysMenu)
}

// CreateWorkoutRecord は完了したワークアウト結果をDBに保存します
func CreateWorkoutRecord(c echo.Context) error {
	// リクエストボディからデータを受け取る構造体を定義（modelsから引用）
	// ※ここでは簡略化のためinterface{}を使用していますが、実際はmodels.WorkoutSessionを使います
	var record map[string]interface{}
	
	if err := c.Bind(&record); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// ※ここでGORMを使ってPostgreSQLにINSERTする処理が入ります

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Workout successfully recorded",
		"status":  "success",
	})
}