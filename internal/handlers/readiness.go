package handlers

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/labstack/echo/v4"
	"gorm.io/clause"
)

// --- 1. Request DTO ---
type PostReadinessRequest struct {
	InputDate             string  `json:"input_date"`      // ISO format: 2026-05-08
	SleepHours            float64 `json:"sleep_hours"`     // 0-12
	MuscleSoreness        int     `json:"muscle_soreness"` // 0-10
	RunningKmPriorDay     float64 `json:"running_km_prior_day"`
}

// --- 2. Response DTO ---
type ReadinessResponse struct {
	InputDate        string  `json:"input_date"`
	ReadinessScore   int     `json:"readiness_score"`   // 0-100
	DeloadFactor     float64 `json:"deload_factor"`     // 0.7-1.0
	DeloadTargetSets int     `json:"deload_target_sets"`
	Recommendation   string  `json:"recommendation"`
}

// --- 3. Algorithm: Calculate Readiness Score ---
// Reference: PRODUCT_SPECIFICATION.md#7-algorithm-reference
func CalculateReadinessScore(sleepHours float64, soreness int, runningKm float64) (score int, deloadFactor float64) {
	score = 100

	// Sleep impact: (8 - sleep_hours) × 5
	// Missing 1 hour of sleep = -5 points
	sleepImpact := math.Max(0, (8.0-sleepHours)*5.0)
	score -= int(sleepImpact)

	// Soreness impact: soreness × 2
	// Each point on 0-10 scale = -2 points
	sorenessImpact := soreness * 2
	score -= sorenessImpact

	// Running interference: running_km × 1.5
	// Each km of running = -1.5 points
	runningImpact := int(runningKm * 1.5)
	score -= runningImpact

	// Clamp to 50-100 range (never drop below 50)
	if score < 50 {
		score = 50
	} else if score > 100 {
		score = 100
	}

	// Deload factor determination
	if score >= 85 {
		deloadFactor = 1.0 // Full volume
	} else if score >= 70 {
		deloadFactor = 0.85 // Reduce by 15%
	} else {
		deloadFactor = 0.70 // Reduce by 30%
	}

	return score, deloadFactor
}

// --- 4. GET /api/readiness/:date ---
// Fetch readiness data for a specific date
func GetReadiness(c echo.Context) error {
	db := database.DB
	dateStr := c.Param("date") // 2026-05-08 format

	// Parse date
	inputDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid date format; use YYYY-MM-DD",
		})
	}

	// Hardcoded user for now (Phase 2: add JWT auth)
	var user models.User
	db.FirstOrCreate(&user, models.User{Email: "test@athenetic.app", Name: "Test User"})

	var readiness models.DailyReadinessInput
	result := db.Where("user_id = ? AND DATE(input_date) = ?", user.ID, inputDate.Format("2006-01-02")).
		First(&readiness)

	if result.RowsAffected == 0 {
		// No data for this date
		return c.JSON(http.StatusNotFound, map[string]string{
			"message": "No readiness data found for this date",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"input_date":      readiness.InputDate.Format("2006-01-02"),
		"sleep_hours":     readiness.SleepHours,
		"muscle_soreness": readiness.MuscleSoreness,
		"running_km":      readiness.RunningKmPriorDay,
		"readiness_score": readiness.ReadinessScore,
		"deload_factor":   readiness.DeloadFactor,
	})
}

// --- 5. POST /api/readiness ---
// Submit daily readiness inputs and calculate score
func PostReadiness(c echo.Context) error {
	var req PostReadinessRequest
	if err := c.Bind(&req); err != nil {
		fmt.Println("❌ [Readiness POST Error] Request parsing failed:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Request format error"})
	}

	// Parse input_date
	inputDate, err := time.Parse("2006-01-02", req.InputDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid date format; use YYYY-MM-DD",
		})
	}

	// Validate inputs
	if req.SleepHours < 0 || req.SleepHours > 12 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "sleep_hours must be 0-12",
		})
	}
	if req.MuscleSoreness < 0 || req.MuscleSoreness > 10 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "muscle_soreness must be 0-10",
		})
	}
	if req.RunningKmPriorDay < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "running_km_prior_day cannot be negative",
		})
	}

	db := database.DB

	// Hardcoded user for now (Phase 2: add JWT auth)
	var user models.User
	db.FirstOrCreate(&user, models.User{Email: "test@athenetic.app", Name: "Test User"})

	// Calculate readiness
	score, deloadFactor := CalculateReadinessScore(
		req.SleepHours,
		req.MuscleSoreness,
		req.RunningKmPriorDay,
	)

	// Determine default target sets (typically 3 per exercise)
	defaultTargetSets := 3
	deloadTargetSets := int(float64(defaultTargetSets) * deloadFactor)

	// Build recommendation message
	recommendation := ""
	if score >= 85 {
		recommendation = "You're in peak condition! Go for full volume 💪"
	} else if score >= 70 {
		recommendation = fmt.Sprintf("Take it easy; reduce volume by 15%% (%d → %d sets)", defaultTargetSets, deloadTargetSets)
	} else {
		recommendation = fmt.Sprintf("Prioritize recovery; reduce volume by 30%% (%d → %d sets)", defaultTargetSets, deloadTargetSets)
	}

	// Save to database
	// First, try to update existing record for this day
	readiness := models.DailyReadinessInput{
		UserID:            user.ID,
		InputDate:         inputDate,
		SleepHours:        req.SleepHours,
		MuscleSoreness:    req.MuscleSoreness,
		RunningKmPriorDay: req.RunningKmPriorDay,
		ReadinessScore:    score,
		DeloadFactor:      deloadFactor,
	}

	// Use Upsert (update if exists, create if not)
	result := db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&readiness)

	if result.Error != nil {
		fmt.Printf("❌ [Readiness POST Error] DB save failed: %v\n", result.Error)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save readiness data",
		})
	}

	fmt.Printf("✅ [Readiness POST Success] Score: %d, Deload: %.2f\n", score, deloadFactor)

	return c.JSON(http.StatusCreated, ReadinessResponse{
		InputDate:        inputDate.Format("2006-01-02"),
		ReadinessScore:   score,
		DeloadFactor:     deloadFactor,
		DeloadTargetSets: deloadTargetSets,
		Recommendation:   recommendation,
	})
}
