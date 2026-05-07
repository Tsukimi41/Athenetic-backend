// internal/routes/routes.go
package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/Tsukimi41/Athenetic-backend/internal/handlers"
)

// SetupRoutes はEchoインスタンスにルーティングを登録します
func SetupRoutes(e *echo.Echo) {
	// APIのバージョニングを行うグループ
	api := e.Group("/api/v1")

	// ワークアウト関連のエンドポイント
	// 例: GET /api/v1/workouts/today -> 今日のメニューを取得
	api.GET("/workouts/today", handlers.GetTodaysWorkout)
	
	// 例: POST /api/v1/workouts -> ワークアウト結果を保存
	api.POST("/workouts", handlers.CreateWorkoutRecord)
}