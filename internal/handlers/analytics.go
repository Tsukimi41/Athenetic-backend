package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/labstack/echo/v4"
)

type volumeRow struct {
	WeekStart  time.Time `gorm:"column:week_start"`
	VolumeLoad float64   `gorm:"column:volume_load"`
	TotalSets  int       `gorm:"column:total_sets"`
	RPEAvg     float64   `gorm:"column:rpe_avg"`
}

type multiVolumeRow struct {
	WeekStart  time.Time `gorm:"column:week_start"`
	Muscle     string    `gorm:"column:muscle"`
	VolumeLoad float64   `gorm:"column:volume_load"`
}

func GetVolumeProgression(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	muscleGroup := strings.ToLower(strings.TrimSpace(c.QueryParam("muscle_group")))
	if muscleGroup == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "muscle_group is required"})
	}
	if muscleGroup != "all" {
		if _, err := parseMuscleGroup(muscleGroup); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid muscle_group"})
		}
	}

	weeks := parseWeeks(c.QueryParam("weeks"), 12, 1, 52)
	start := startOfWeek(time.Now().UTC()).AddDate(0, 0, -7*(weeks-1))

	db := database.DB
	if muscleGroup == "all" {
		query := db.Model(&models.WorkoutSet{}).
			Select("DATE_TRUNC('week', workout_sets.created_at) AS week_start, LOWER(e.target_muscle) AS muscle, COALESCE(SUM((CASE WHEN e.is_bodyweight THEN COALESCE(NULLIF(workout_sets.weight, 0), u.body_weight, 0) ELSE workout_sets.weight END) * workout_sets.reps), 0) AS volume_load").
			Joins("JOIN workout_sessions ws ON ws.id = workout_sets.session_id").
			Joins("JOIN exercises e ON e.id = workout_sets.exercise_id").
			Joins("JOIN users u ON u.id = ws.user_id").
			Where("ws.user_id = ?", userID).
			Where("workout_sets.created_at >= ?", start).
			Group("week_start, muscle").
			Order("week_start ASC")

		var rows []multiVolumeRow
		if err := query.Scan(&rows).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load progression"})
		}

		rowMap := map[string]map[string]float64{}
		for _, row := range rows {
			key := row.WeekStart.Format("2006-01-02")
			if rowMap[key] == nil {
				rowMap[key] = map[string]float64{}
			}
			rowMap[key][row.Muscle] = row.VolumeLoad
		}

		response := make([]map[string]interface{}, 0, weeks)
		for i := 0; i < weeks; i++ {
			weekStart := start.AddDate(0, 0, 7*i)
			key := weekStart.Format("2006-01-02")
			row := rowMap[key]

			item := map[string]interface{}{
				"week":       i + 1,
				"week_start": key,
				"chest":      0.0,
				"back":       0.0,
				"legs":       0.0,
			}
			if row != nil {
				if value, ok := row["chest"]; ok {
					item["chest"] = value
				}
				if value, ok := row["back"]; ok {
					item["back"] = value
				}
				if value, ok := row["legs"]; ok {
					item["legs"] = value
				}
			}

			response = append(response, item)
		}

		return c.JSON(http.StatusOK, response)
	}

	query := db.Model(&models.WorkoutSet{}).
		Select("DATE_TRUNC('week', workout_sets.created_at) AS week_start, COALESCE(SUM((CASE WHEN e.is_bodyweight THEN COALESCE(NULLIF(workout_sets.weight, 0), u.body_weight, 0) ELSE workout_sets.weight END) * workout_sets.reps), 0) AS volume_load, COUNT(workout_sets.id) AS total_sets, COALESCE(AVG(workout_sets.rpe), 0) AS rpe_avg").
		Joins("JOIN workout_sessions ws ON ws.id = workout_sets.session_id").
		Joins("JOIN exercises e ON e.id = workout_sets.exercise_id").
		Joins("JOIN users u ON u.id = ws.user_id").
		Where("ws.user_id = ?", userID).
		Where("workout_sets.created_at >= ?", start).
		Where("LOWER(e.target_muscle) = LOWER(?)", muscleGroup).
		Group("week_start").
		Order("week_start ASC")

	var rows []volumeRow
	if err := query.Scan(&rows).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load progression"})
	}

	rowMap := map[string]volumeRow{}
	for _, row := range rows {
		rowMap[row.WeekStart.Format("2006-01-02")] = row
	}

	response := make([]map[string]interface{}, 0, weeks)
	for i := 0; i < weeks; i++ {
		weekStart := start.AddDate(0, 0, 7*i)
		key := weekStart.Format("2006-01-02")
		row := rowMap[key]

		response = append(response, map[string]interface{}{
			"week":        i + 1,
			"week_start":  key,
			"volume_load": row.VolumeLoad,
			"target":      0,
			"sets":        row.TotalSets,
			"rpe_avg":     row.RPEAvg,
		})
	}

	return c.JSON(http.StatusOK, response)
}

func GetProgressSummary(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	muscleGroup := c.QueryParam("muscle_group")
	if muscleGroup != "" {
		if _, err := parseMuscleGroup(muscleGroup); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid muscle_group"})
		}
	}

	now := time.Now().UTC()
	currentWeekStart := startOfWeek(now)
	priorWeekStart := currentWeekStart.AddDate(0, 0, -7)

	currentLoad := volumeForRange(userID, currentWeekStart, currentWeekStart.AddDate(0, 0, 7), muscleGroup)
	priorLoad := volumeForRange(userID, priorWeekStart, currentWeekStart, muscleGroup)

	delta := currentLoad - priorLoad
	deltaPercent := 0.0
	if priorLoad > 0 {
		deltaPercent = (delta / priorLoad) * 100
	}

	status := "Stable training volume"
	if deltaPercent >= 5 {
		status = "Strong progress!"
	} else if deltaPercent <= -5 {
		status = "Recovery emphasis recommended"
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_week_load": currentLoad,
		"prior_week_load":   priorLoad,
		"delta":             delta,
		"delta_percent":     deltaPercent,
		"status":            status,
	})
}

func volumeForRange(userID interface{}, start time.Time, end time.Time, muscleGroup string) float64 {
	db := database.DB
	query := db.Model(&models.WorkoutSet{}).
		Select("COALESCE(SUM((CASE WHEN e.is_bodyweight THEN COALESCE(NULLIF(workout_sets.weight, 0), u.body_weight, 0) ELSE workout_sets.weight END) * workout_sets.reps), 0) AS volume_load").
		Joins("JOIN workout_sessions ws ON ws.id = workout_sets.session_id").
		Joins("JOIN exercises e ON e.id = workout_sets.exercise_id").
		Joins("JOIN users u ON u.id = ws.user_id").
		Where("ws.user_id = ?", userID).
		Where("workout_sets.created_at >= ? AND workout_sets.created_at < ?", start, end)

	if muscleGroup != "" {
		query = query.Where("LOWER(e.target_muscle) = LOWER(?)", muscleGroup)
	}

	var result struct {
		VolumeLoad float64 `gorm:"column:volume_load"`
	}
	_ = query.Scan(&result)

	return result.VolumeLoad
}

func parseWeeks(input string, defaultValue int, min int, max int) int {
	if input == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(input)
	if err != nil {
		return defaultValue
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func startOfWeek(t time.Time) time.Time {
	start := t.Truncate(24 * time.Hour)
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	return start
}
