package database

import (
	"log"
	"os"

	// 先ほど作成したmodelsパッケージをインポート
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// グローバル変数としてDBインスタンスを保持し、どこからでもアクセスできるようにする
var DB *gorm.DB

func ConnectDB() {
	// docker-compose.yml で設定した情報と完全に一致させる
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=athenetic_user password=athenetic_password dbname=athenetic_db port=5432 sslmode=disable TimeZone=Asia/Tokyo"
	}
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database!\n", err)
	}

	log.Println("Database connection successfully opened")

	// 魔法の機能「AutoMigrate」
	// Goの構造体(Struct)を読み込み、足りないテーブルやカラムをPostgreSQL上に自動で生成・更新します
	err = db.AutoMigrate(
		&models.User{},
		&models.Exercise{},
		&models.WorkoutSession{},
		&models.WorkoutSet{},
		&models.DailyReadinessInput{},
		&models.NutritionLog{},
		&models.RefreshToken{},
	)
	if err != nil {
		log.Fatal("Failed to auto migrate database!\n", err)
	}

	// ユニーク制約を追加 (user_id, input_date)
	db.Migrator().CreateConstraint(&models.DailyReadinessInput{}, "uni_user_date")
	if !db.Migrator().HasConstraint(&models.DailyReadinessInput{}, "uni_user_date") {
		db.Exec(`ALTER TABLE daily_readiness_inputs ADD CONSTRAINT uni_user_date UNIQUE(user_id, input_date)`)
	}

	log.Println("Database Auto Migration completed")

	// Seed exercises (Phase 1: Chest, Back, Legs)
	seedExercises(db)

	// 接続が成功したらグローバル変数に格納
	DB = db
}

// seedExercises: 種目マスターデータを初期化
func seedExercises(db *gorm.DB) {
	exercises := []models.Exercise{
		// Chest (Push)
			{Name: "Barbell Bench Press", TargetMuscle: models.Chest, DefaultTargetSets: 3, IsBodyweight: false},
			{Name: "Decline Push-up", TargetMuscle: models.Chest, DefaultTargetSets: 3, IsBodyweight: true},
			{Name: "Archer Push-up", TargetMuscle: models.Chest, DefaultTargetSets: 3, IsBodyweight: true},
			{Name: "Dumbbell Flyes", TargetMuscle: models.Chest, DefaultTargetSets: 3, IsBodyweight: false},

		// Back (Pull)
			{Name: "Pull-ups", TargetMuscle: models.Back, DefaultTargetSets: 3, IsBodyweight: true},
			{Name: "Barbell Rows", TargetMuscle: models.Back, DefaultTargetSets: 3, IsBodyweight: false},
			{Name: "Lat Pulldowns", TargetMuscle: models.Back, DefaultTargetSets: 3, IsBodyweight: false},
			{Name: "Reverse Flyes", TargetMuscle: models.Back, DefaultTargetSets: 3, IsBodyweight: false},

		// Legs
			{Name: "Barbell Squats", TargetMuscle: models.Legs, DefaultTargetSets: 3, IsBodyweight: false},
			{Name: "Leg Press", TargetMuscle: models.Legs, DefaultTargetSets: 3, IsBodyweight: false},
			{Name: "Deadlifts", TargetMuscle: models.Legs, DefaultTargetSets: 3, IsBodyweight: false},
			{Name: "Bulgarian Split Squats", TargetMuscle: models.Legs, DefaultTargetSets: 3, IsBodyweight: true},

		// Shoulders
			{Name: "Overhead Press", TargetMuscle: models.Shoulders, DefaultTargetSets: 3, IsBodyweight: false},
			{Name: "Lateral Raises", TargetMuscle: models.Shoulders, DefaultTargetSets: 3, IsBodyweight: false},
	}

	// Only insert if not already present
	for _, ex := range exercises {
		var existing models.Exercise
		// Case-insensitive check
		result := db.Where("LOWER(name) = LOWER(?)", ex.Name).First(&existing)
		if result.RowsAffected == 0 {
			db.Create(&ex)
			log.Printf("✅ Seeded exercise: %s (%s)\n", ex.Name, ex.TargetMuscle)
		}
	}
}