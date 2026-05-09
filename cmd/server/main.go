// cmd/server/main.go
package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/routes"
)

func main() {
	// サーバー起動前にデータベースに接続し、テーブルを自動生成する
	database.ConnectDB()
	// Echoインスタンスの作成
	e := echo.New()
	e.HideBanner = true

	// ここでミドルウェアの設定やルーティングのセットアップを行います

	// ミドルウェアの設定
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())  // リクエストのログ出力
	e.Use(middleware.Recover()) // パニック（クラッシュ）時の自動復旧
	e.Use(middleware.BodyLimit("2M"))
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 15 * time.Second,
	}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'none'",
	}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(30)))
	
	// CORS設定（Next.jsからのアクセスを許可するため必須）
	allowOrigins := strings.Split(strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")), ",")
	if len(allowOrigins) == 0 || (len(allowOrigins) == 1 && allowOrigins[0] == "") {
		allowOrigins = []string{
			"http://localhost:3000",     // Local dev
			"http://127.0.0.1:3000",
			"https://athenetic.vercel.app", // Production
		}
	}

	for i, origin := range allowOrigins {
		allowOrigins[i] = strings.TrimSpace(origin)
	}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     allowOrigins,
		AllowCredentials: true,
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		MaxAge:       3600,
	}))

	// ルーティングのセットアップ
	routes.SetupRoutes(e)

	// サーバー起動 (ポート8080)
	log.Println("Starting Athenetic API server on port 8080...")
	e.Logger.Fatal(e.Start(":8080"))
}