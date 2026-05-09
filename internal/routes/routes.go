// internal/routes/routes.go
package routes

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/Tsukimi41/Athenetic-backend/internal/handlers"
	authmw "github.com/Tsukimi41/Athenetic-backend/internal/middleware"
)

// SetupRoutes はEchoインスタンスにルーティングを登録します
func SetupRoutes(e *echo.Echo) {
	// APIのバージョニングを行うグループ
	api := e.Group("/api/v1")

	// Auth endpoints
	auth := api.Group("/auth")
	auth.Use(echomw.RateLimiter(echomw.NewRateLimiterMemoryStore(5)))
	auth.POST("/signup", handlers.Signup)
	auth.POST("/login", handlers.Login)
	auth.POST("/refresh", handlers.Refresh)
	auth.GET("/me", handlers.Me, authmw.JWTAuth)
	auth.PATCH("/me", handlers.UpdateMe, authmw.JWTAuth)
	auth.POST("/logout", handlers.Logout, authmw.JWTAuth)

	// Protected endpoints
	protected := api.Group("")
	protected.Use(authmw.JWTAuth)

	// ワークアウト関連のエンドポイント
	// 例: GET /api/v1/workouts/today -> 今日のメニューを取得
	protected.GET("/workouts/today", handlers.GetTodaysWorkout)
	
	// 例: POST /api/v1/workouts -> ワークアウト結果を保存
	protected.POST("/workouts", handlers.CreateWorkoutRecord)

	protected.GET("/analytics/volume", handlers.GetWeeklyVolume)
	protected.GET("/analytics/volume-progression", handlers.GetVolumeProgression)
	protected.GET("/analytics/progress", handlers.GetProgressSummary)

	// Readiness (コンディション管理) - Phase 1
	protected.POST("/readiness", handlers.PostReadiness)
	protected.GET("/readiness/:date", handlers.GetReadiness)

	// Sessions and sets
	protected.GET("/sessions/:date", handlers.GetSessionsByDate)
	protected.POST("/sessions", handlers.CreateSession)
	protected.POST("/sets", handlers.CreateSet)
	protected.GET("/previous-set/:exercise_id", handlers.GetPreviousSet)

	// Exercises catalog
	protected.GET("/exercises", handlers.ListExercises)

	// Nutrition
	protected.POST("/nutrition/log", handlers.PostNutritionLog)
	protected.GET("/nutrition/summary/:date", handlers.GetNutritionSummary)
}