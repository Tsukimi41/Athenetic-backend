package handlers

import (
	"net/http"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/labstack/echo/v4"
)

type NutritionLogRequest struct {
	LogDate  string  `json:"log_date"`
	FoodName string  `json:"food_name"`
	ProteinG float64 `json:"protein_g"`
	CarbsG   float64 `json:"carbs_g"`
	FatG     float64 `json:"fat_g"`
}

func PostNutritionLog(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req NutritionLogRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.LogDate == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "log_date is required"})
	}

	logDate, err := time.Parse("2006-01-02", req.LogDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid log_date"})
	}

	if req.ProteinG < 0 || req.CarbsG < 0 || req.FatG < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "macros must be positive"})
	}

	log := models.NutritionLog{
		UserID:   userID,
		LogDate:  logDate,
		FoodName: req.FoodName,
		ProteinG: req.ProteinG,
		CarbsG:   req.CarbsG,
		FatG:     req.FatG,
		CreatedAt: time.Now(),
	}

	db := database.DB
	if err := db.Create(&log).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to log nutrition"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "nutrition logged"})
}

func GetNutritionSummary(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	dateStr := c.Param("date")
	logDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid date"})
	}

	db := database.DB

	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user not found"})
	}

	var totals struct {
		Protein float64
		Carbs   float64
		Fat     float64
	}

	if err := db.Model(&models.NutritionLog{}).
		Select("COALESCE(SUM(protein_g), 0) AS protein, COALESCE(SUM(carbs_g), 0) AS carbs, COALESCE(SUM(fat_g), 0) AS fat").
		Where("user_id = ? AND DATE(log_date) = ?", userID, logDate.Format("2006-01-02")).
		Scan(&totals).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load nutrition summary"})
	}

	proteinTarget := user.BodyWeight * 1.6
	totalCalories := (totals.Protein * 4) + (totals.Carbs * 4) + (totals.Fat * 9)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"date":             logDate.Format("2006-01-02"),
		"body_weight_kg":   user.BodyWeight,
		"protein_target":   proteinTarget,
		"protein_logged":   totals.Protein,
		"carbs_logged":     totals.Carbs,
		"fat_logged":       totals.Fat,
		"total_calories":   totalCalories,
	})
}
