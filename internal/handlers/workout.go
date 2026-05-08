package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	//"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)


// GetTodaysWorkout はDBの過去データから漸進性過負荷を計算し、本日のメニューを動的に生成します
func GetTodaysWorkout(c echo.Context) error {
	db := database.DB

	// 1. ユーザーの取得（本来はJWT認証から取得）
	var user models.User
	db.FirstOrCreate(&user, models.User{Email: "test@athenetic.app", Name: "Test User"})

	// 2. 本日のコンディション（レディネススコア）の仮計算
	// ※ゆくゆくは睡眠データや前日のランニング距離（干渉効果の考慮）から算出
	readinessScore := 92

	// 3. 今日のメニュー構成（曜日ごとに部位を判定するロジックの基礎）
	// 今回はデモとして「Upper Body Push」の種目をDBから検索、または作成
	exerciseNames := []string{"Decline Push-up", "Archer Push-up"}
	var exercisesData []map[string]interface{}

	for _, name := range exerciseNames {
		var exercise models.Exercise
		db.FirstOrCreate(&exercise, models.Exercise{
			Name:         name,
			TargetMuscle: models.Chest,
			IsBodyweight: true,
		})

		// 4. 頭脳の中核：前回の記録をDBから取得し、漸進性過負荷を計算
		var lastSet models.WorkoutSet
		// 直近の完了済みセットを取得
		result := db.Where("exercise_id = ? AND is_completed = ?", exercise.ID, true).
			Order("created_at desc").First(&lastSet)

		targetSets := 3 // デフォルトのセット数
		targetReps := 10 // デフォルトの回数
		historyText := "初回トライアル: 目安10 reps"

		if result.RowsAffected > 0 {
			// 前回の記録がある場合、アルゴリズムに基づいて目標回数を再計算
			targetReps = lastSet.Reps

			if lastSet.RPE <= 8.0 {
				targetReps += 2 // 余裕があったので2回増やす
			} else if lastSet.RPE <= 9.0 {
				targetReps += 1 // 適切に追い込めたので1回増やす
			}
			// RPEが9.0を超えていた場合は回数をキープしてフォーム改善に努める

			historyText = fmt.Sprintf("前回: %d reps (RPE %.1f)", lastSet.Reps, lastSet.RPE)
		}

		// 計算結果をスライスに追加
		exercisesData = append(exercisesData, map[string]interface{}{
			"id":          exercise.ID,
			"name":        exercise.Name,
			"target_sets": targetSets,
			"target_reps": targetReps, // 新たに追加された動的な目標回数
			"history":     historyText,
		})
	}

	// 5. 最終的なスケジュールをフロントエンドに送信
	todaysMenu := map[string]interface{}{
		"date":            time.Now().Format("2006-01-02"),
		"title":           "Upper Body Push",
		"readiness_score": readinessScore,
		"exercises":       exercisesData,
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

// GetWeeklyVolume は過去7日間の部位ごとのセット数を集計します
func GetWeeklyVolume(c echo.Context) error {
	db := database.DB

	// 1. ダミーユーザーの取得（本来はログイン情報から）
	var user models.User
	if err := db.Where("email = ?", "test@athenetic.app").First(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "ユーザーが見つかりません"})
	}

	// 2. 7日前の時間を計算
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	// 3. 集計結果を受け取るための構造体
	type Result struct {
		TargetMuscle string
		TotalSets    int
	}
	var results []Result

	// 4. SQLの魔法：WorkoutSetとExerciseを結合し、7日以内の完了セットを部位ごとにカウント
	db.Table("workout_sets").
		Select("exercises.target_muscle, count(workout_sets.id) as total_sets").
		Joins("left join exercises on exercises.id = workout_sets.exercise_id").
		Joins("left join workout_sessions on workout_sessions.id = workout_sets.session_id").
		Where("workout_sessions.user_id = ? AND workout_sets.is_completed = ? AND workout_sets.created_at >= ?", user.ID, true, sevenDaysAgo).
		Group("exercises.target_muscle").
		Scan(&results)

	// 5. フロントエンドに返すためのMapを作成（初期値は0）
	volumeData := map[string]int{
		"Chest": 0,
		"Back":  0,
		"Legs":  0,
	}

	// 6. SQLの結果をMapに上書き
	for _, r := range results {
		volumeData[r.TargetMuscle] = r.TotalSets
	}

	return c.JSON(http.StatusOK, volumeData)
}