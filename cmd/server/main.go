// cmd/server/main.go
package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/Tsukimi41/Athenetic-backend/internal/routes"
)

func main() {
	// Echoインスタンスの作成
	e := echo.New()

	// ミドルウェアの設定
	e.Use(middleware.Logger())  // リクエストのログ出力
	e.Use(middleware.Recover()) // パニック（クラッシュ）時の自動復旧
	
	// CORS設定（Next.jsからのアクセスを許可するため必須）
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000"}, // Next.jsのローカルURL
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// ルーティングのセットアップ
	routes.SetupRoutes(e)

	// サーバー起動 (ポート8080)
	log.Println("Starting Athenetic API server on port 8080...")
	e.Logger.Fatal(e.Start(":8080"))
}