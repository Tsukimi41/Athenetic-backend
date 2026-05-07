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
	IsBodyweight bool         `gorm:"default:true"`
	CreatedAt    time.Time
}

// 3. WorkoutSessions (セッション・ログテーブル)
type WorkoutSession struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         uuid.UUID `gorm:"type:uuid;index;not null"`
	Title          string    `gorm:"type:varchar(100)"` // 例: "Upper Body Push"
	StartTime      time.Time `gorm:"not null"`
	EndTime        *time.Time
	ReadinessScore int       `gorm:"type:int"` // コンディションスコア
	CreatedAt      time.Time

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
	Weight    float64 `gorm:"type:numeric(6,2);default:0"`

	// スポーツ科学的パラメータ
	RPE      float64 `gorm:"type:numeric(3,1)"` // 主観的運動強度
	TUT      int     `gorm:"type:int"`          // 緊張下時間(秒)
	RestTime int     `gorm:"type:int"`          // 休憩時間(秒)

	IsCompleted bool `gorm:"default:false"`
	CreatedAt   time.Time
}