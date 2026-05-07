package database

import (
	"log"

	// 先ほど作成したmodelsパッケージをインポート
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// グローバル変数としてDBインスタンスを保持し、どこからでもアクセスできるようにする
var DB *gorm.DB

func ConnectDB() {
	// docker-compose.yml で設定した情報と完全に一致させる
	dsn := "host=localhost user=athenetic_user password=athenetic_password dbname=athenetic_db port=5432 sslmode=disable TimeZone=Asia/Tokyo"
	
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
	)
	if err != nil {
		log.Fatal("Failed to auto migrate database!\n", err)
	}

	log.Println("Database Auto Migration completed")

	// 接続が成功したらグローバル変数に格納
	DB = db
}