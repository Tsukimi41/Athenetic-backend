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

	var user models.User
	db.FirstOrCreate(&user, models.User{Email: "test@athenetic.app", Name: "Test User"})

	// Query parameter: muscle_group (default: "chest")
	muscleGroup := c.QueryParam("muscle_group")
	if muscleGroup == "" {
		muscleGroup = "chest"
	}

	readinessScore := 92
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
		result := db.Where("exercise_id = ? AND is_completed = ?", exercise.ID, true).
			Order("created_at desc").First(&lastSet)

		targetSets := 3
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
}

// --- 2. POST: セットの保存 ---
func CreateWorkoutRecord(c echo.Context) error {
	var req CompleteSetRequest
	if err := c.Bind(&req); err != nil {
		fmt.Println("❌ [POSTエラー] リクエスト解析失敗:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "リクエスト形式エラー"})
	}

	db := database.DB
	var user models.User
	db.FirstOrCreate(&user, models.User{Email: "test@athenetic.app", Name: "Test User"})

	// 日付による検索バグを防ぐため、シンプルな文字列検索に統一
	var session models.WorkoutSession
	if err := db.Where("user_id = ? AND title = ?", user.ID, "Daily Workout").First(&session).Error; err != nil {
		session = models.WorkoutSession{
			UserID:         user.ID,
			Title:          "Daily Workout",
			StartTime:      time.Now(),
			ReadinessScore: 90,
		}
		db.Create(&session)
	}

	var exercise models.Exercise
	db.Where("name = ?", req.ExerciseName).FirstOrCreate(&exercise, models.Exercise{
		Name:         req.ExerciseName,
		TargetMuscle: models.Chest,
		IsBodyweight: true,
	})

	workoutSet := models.WorkoutSet{
		SessionID:   session.ID,
		ExerciseID:  exercise.ID,
		SetNumber:   req.SetNumber,
		Reps:        req.Reps,
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

	// 1. 複雑な条件（セッションIDや日付）をすべて取り払い、とにかく完了済みの全セットを強制取得
	var allSets []models.WorkoutSet
	db.Where("is_completed = ?", true).Find(&allSets)

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