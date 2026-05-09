package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/labstack/echo/v4"
)

// --- 1. GET: 今日のメニュー ---
func GetTodaysWorkout(c echo.Context) error {
	db := database.DB

	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	// Query parameter: muscle_group (default: "chest")
	muscleGroup := c.QueryParam("muscle_group")
	if muscleGroup == "" {
		muscleGroup = "chest"
	}
	muscleGroup = strings.ToLower(muscleGroup)

	readinessScore := 90
	deloadFactor := 1.0
	today := time.Now().Format("2006-01-02")
	var readiness models.DailyReadinessInput
	if err := db.Where("user_id = ? AND DATE(input_date) = ?", userID, today).First(&readiness).Error; err == nil {
		readinessScore = readiness.ReadinessScore
		deloadFactor = readiness.DeloadFactor
	}
	var exercises []models.Exercise
	
	// Fetch exercises for the specified muscle group (case-insensitive)
	// CRITICAL RULE: Always use LOWER() for string comparisons
	db.Where("LOWER(target_muscle) = LOWER(?)", muscleGroup).Find(&exercises)

	// If no exercises found, return error
	if len(exercises) == 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"date":            time.Now().Format("2006-01-02"),
			"title":           "No exercises for " + muscleGroup,
			"readiness_score": readinessScore,
			"exercises":       []map[string]interface{}{},
		})
	}

	var exercisesData []map[string]interface{}

	for _, exercise := range exercises {
		var lastSet models.WorkoutSet
		result := db.Model(&models.WorkoutSet{}).
			Joins("JOIN workout_sessions ws ON ws.id = workout_sets.session_id").
			Where("ws.user_id = ?", userID).
			Where("workout_sets.exercise_id = ? AND workout_sets.is_completed = ?", exercise.ID, true).
			Order("workout_sets.created_at desc").
			First(&lastSet)

		defaultSets := exercise.DefaultTargetSets
		if defaultSets < 1 {
			defaultSets = 3
		}
		targetSets := int(float64(defaultSets) * deloadFactor)
		if targetSets < 1 {
			targetSets = 1
		}
		targetReps := 10
		historyText := "First attempt: aim for 10 reps"

		if result.RowsAffected > 0 {
			targetReps = lastSet.Reps
			if lastSet.RPE <= 8.0 {
				targetReps += 2
			} else if lastSet.RPE <= 9.0 {
				targetReps += 1
			}
			historyText = fmt.Sprintf("Previous: %d reps (RPE %.1f)", lastSet.Reps, lastSet.RPE)
		}

		exercisesData = append(exercisesData, map[string]interface{}{
			"id":          exercise.ID,
			"name":        exercise.Name,
			"target_sets": targetSets,
			"target_reps": targetReps,
			"history":     historyText,
		})
	}

	title := map[string]string{
		"chest":      "Upper Body Push",
		"back":       "Upper Body Pull",
		"legs":       "Lower Body Strength",
		"shoulders":  "Shoulder Development",
		"core":       "Core Stability",
	}[muscleGroup]

	if title == "" {
		title = "Strength Training"
	}

	todaysMenu := map[string]interface{}{
		"date":            time.Now().Format("2006-01-02"),
		"title":           title,
		"muscle_group":    muscleGroup,
		"readiness_score": readinessScore,
		"exercises":       exercisesData,
	}

	return c.JSON(http.StatusOK, todaysMenu)
}

type CompleteSetRequest struct {
	ExerciseName string  `json:"exercise_name"`
	SetNumber    int     `json:"set_number"`
	Reps         int     `json:"reps"`
	RPE          float64 `json:"rpe"`
	WeightKg     float64 `json:"weight_kg"`
	TargetReps   int     `json:"target_reps"`
}

// --- 2. POST: セットの保存 ---
func CreateWorkoutRecord(c echo.Context) error {
	var req CompleteSetRequest
	if err := c.Bind(&req); err != nil {
		fmt.Println("❌ [POSTエラー] リクエスト解析失敗:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "リクエスト形式エラー"})
	}
	if req.WeightKg < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "weight_kg must be positive"})
	}
	if req.TargetReps <= 0 {
		req.TargetReps = req.Reps
	}

	db := database.DB

	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var exercise models.Exercise
	db.Where("LOWER(name) = LOWER(?)", req.ExerciseName).FirstOrCreate(&exercise, models.Exercise{
		Name:              req.ExerciseName,
		TargetMuscle:      models.Chest,
		DefaultTargetSets: 3,
		IsBodyweight:      true,
	})

	weight := req.WeightKg
	if exercise.IsBodyweight && weight <= 0 {
		var user models.User
		if err := db.First(&user, "id = ?", userID).Error; err == nil && user.BodyWeight > 0 {
			weight = user.BodyWeight
		}
	}

	readinessScore := 90
	var readiness models.DailyReadinessInput
	if err := db.Where("user_id = ? AND DATE(input_date) = ?", userID, time.Now().Format("2006-01-02")).First(&readiness).Error; err == nil {
		readinessScore = readiness.ReadinessScore
	}

	// Ensure a session exists for today and muscle group
	var session models.WorkoutSession
	if err := db.Where("user_id = ? AND DATE(session_date) = ? AND LOWER(muscle_group) = LOWER(?)", userID, time.Now().Format("2006-01-02"), exercise.TargetMuscle).First(&session).Error; err != nil {
		session = models.WorkoutSession{
			UserID:         userID,
			Title:          "Daily Workout",
			SessionDate:    time.Now(),
			MuscleGroup:    exercise.TargetMuscle,
			StartTime:      time.Now(),
			ReadinessScore: readinessScore,
		}
		db.Create(&session)
	}

	workoutSet := models.WorkoutSet{
		SessionID:   session.ID,
		ExerciseID:  exercise.ID,
		SetNumber:   req.SetNumber,
		Reps:        req.Reps,
		TargetReps:  req.TargetReps,
		Weight:      weight,
		RPE:         req.RPE,
		IsCompleted: true,
	}

	if err := db.Create(&workoutSet).Error; err != nil {
		fmt.Println("❌ [POSTエラー] DB保存失敗:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "DB保存エラー"})
	}

	// 成功時のログ出力
	fmt.Printf("✅ [POST成功] %s のセットを保存しました。Reps: %d\n", req.ExerciseName, req.Reps)
	return c.JSON(http.StatusCreated, map[string]interface{}{"message": "Record saved successfully!"})
}

// --- 3. GET: ボリュームの集計（SQLのJOINを排除した確実な純粋Goロジック） ---
// --- 3. GET: ボリュームの集計（究極のデバッグ＆絶対確実版） ---
func GetWeeklyVolume(c echo.Context) error {
	db := database.DB

	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	startOfWeek := time.Now().Truncate(24 * time.Hour)
	for startOfWeek.Weekday() != time.Monday {
		startOfWeek = startOfWeek.AddDate(0, 0, -1)
	}

	// 1. 複雑な条件（セッションIDや日付）をすべて取り払い、とにかく完了済みの全セットを強制取得
	var allSets []models.WorkoutSet
	db.Model(&models.WorkoutSet{}).
		Joins("JOIN workout_sessions ws ON ws.id = workout_sets.session_id").
		Where("ws.user_id = ?", userID).
		Where("workout_sets.is_completed = ?", true).
		Where("workout_sets.created_at >= ?", startOfWeek).
		Find(&allSets)

	fmt.Printf("\n🔍 [分析API起動] DBから取得した完了済み全セット数: %d\n", len(allSets))

	// 2. DB内の全種目を取得
	var exercises []models.Exercise
	db.Find(&exercises)

	// 3. 種目ID -> 部位名 のマップを作成（現在の登録状況をターミナルに暴露します）
	exMap := make(map[string]string)
	for _, ex := range exercises {
		// CRITICAL RULE: Always use LOWER() for string comparisons
		muscleStr := fmt.Sprintf("%v", ex.TargetMuscle)
		muscleStr = strings.ToLower(muscleStr)
		exMap[ex.ID.String()] = muscleStr
		fmt.Printf("🔍 種目: %s -> 登録されている部位: '%s'\n", ex.Name, muscleStr)
	}

	// 4. 純粋にGo言語内でカウント
	volumeData := map[string]int{"Chest": 0, "Back": 0, "Legs": 0}

	for _, s := range allSets {
		muscle := exMap[s.ExerciseID.String()]
		
		// Case-insensitive matching
		if muscle == "chest" {
			volumeData["Chest"]++
		} else if muscle == "back" {
			volumeData["Back"]++
		} else if muscle == "legs" {
			volumeData["Legs"]++
		}
	}

	fmt.Printf("📊 [GET /volume] 最終集計結果: Chest=%d, Back=%d, Legs=%d\n\n", volumeData["Chest"], volumeData["Back"], volumeData["Legs"])

	return c.JSON(http.StatusOK, volumeData)
}