package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 1. Users (ユーザーテーブル)
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	Name         string    `gorm:"type:varchar(100)"`
	BodyWeight   float64   `gorm:"type:numeric(5,2)"` // 自重負荷の計算用
	TargetBodyWeight  float64 `gorm:"type:numeric(5,2)"`
	BodyFatPercentage float64 `gorm:"type:numeric(4,1)"` // 最新の体脂肪率
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// 2. Exercises (種目マスターテーブル)
type TargetMuscle string

const (
	Chest     TargetMuscle = "chest"
	Back      TargetMuscle = "back"
	Legs      TargetMuscle = "legs"
	Shoulders TargetMuscle = "shoulders"
	Core      TargetMuscle = "core"
)

type Exercise struct {
	ID           uuid.UUID    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string       `gorm:"type:varchar(100);uniqueIndex;not null"`
	TargetMuscle TargetMuscle `gorm:"type:varchar(50);not null"`
	DefaultTargetSets int      `gorm:"type:int;default:3"`
	IsBodyweight bool         `gorm:"default:true"`
	CreatedAt    time.Time
}

// 3. WorkoutSessions (セッション・ログテーブル)
type WorkoutSession struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         uuid.UUID `gorm:"type:uuid;index;not null"`
	Title          string    `gorm:"type:varchar(100)"` // 例: "Upper Body Push"
	SessionDate    time.Time `gorm:"type:date;index"`
	MuscleGroup    TargetMuscle `gorm:"type:varchar(50)"`
	StartTime      time.Time `gorm:"not null"`
	EndTime        *time.Time
	ReadinessScore int       `gorm:"type:int"` // コンディションスコア
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// リレーション（1つのセッションが複数のセットを持つ）
	Sets []WorkoutSet `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE;"`
}

// 4. WorkoutSets (セット詳細テーブル)
type WorkoutSet struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SessionID  uuid.UUID `gorm:"type:uuid;index;not null"`
	ExerciseID uuid.UUID `gorm:"type:uuid;index;not null"`

	SetNumber int     `gorm:"type:int;not null"`
	Reps      int     `gorm:"type:int;not null"`
	TargetReps int    `gorm:"type:int"`
	Weight    float64 `gorm:"type:numeric(6,2);default:0"`

	// スポーツ科学的パラメータ
	RPE      float64 `gorm:"type:numeric(3,1)"` // 主観的運動強度
	RIR      int     `gorm:"type:int"`          // Reps in reserve
	TUT      int     `gorm:"type:int"`          // 緊張下時間(秒)
	RestTime int     `gorm:"type:int"`          // 休憩時間(秒)

	IsCompleted bool `gorm:"default:false"`
	CreatedAt   time.Time
}

// 5. DailyReadinessInputs (日次コンディション入力テーブル - Phase 1)
type DailyReadinessInput struct {
	ID                    uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID                uuid.UUID `gorm:"type:uuid;index;not null"`
	InputDate             time.Time `gorm:"type:date;index;not null"`
	SleepHours            float64   `gorm:"type:numeric(3,1)"`           // 睡眠時間 (0-12h)
	MuscleSoreness        int       `gorm:"type:int;default:0"`           // 筋肉痛 (0-10)
	RunningKmPriorDay     float64   `gorm:"type:numeric(5,2);default:0"`  // 前日のランニング距離
	ReadinessScore        int       `gorm:"type:int"`                     // 計算結果: リーディネススコア (0-100)
	DeloadFactor          float64   `gorm:"type:numeric(3,2);default:1"`  // 計算結果: デロード係数 (0.7-1.0)
	CreatedAt             time.Time
	UpdatedAt             time.Time

	// ユニーク制約: ユーザーごとに1日1回のみ
	// (SQLレベルではUNIQUE(user_id, input_date)で実装)
}

// 6. NutritionLogs (栄養ログテーブル - Phase 3)
type NutritionLog struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	LogDate   time.Time `gorm:"type:date;index;not null"`
	FoodName  string    `gorm:"type:varchar(255)"`
	ProteinG  float64   `gorm:"type:numeric(6,2);default:0"`
	CarbsG    float64   `gorm:"type:numeric(6,2);default:0"`
	FatG      float64   `gorm:"type:numeric(6,2);default:0"`
	CreatedAt time.Time
}