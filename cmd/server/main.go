// cmd/server/main.go
package main

import (
	"log"

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

	// ここでミドルウェアの設定やルーティングのセットアップを行います

	// ミドルウェアの設定
	e.Use(middleware.Logger())  // リクエストのログ出力
	e.Use(middleware.Recover()) // パニック（クラッシュ）時の自動復旧
	
	// CORS設定（Next.jsからのアクセスを許可するため必須）
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",     // Local dev
			"http://127.0.0.1:3000",
			"https://athenetic.vercel.app", // Production
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		MaxAge:       3600,
	}))

	// ルーティングのセットアップ
	routes.SetupRoutes(e)

	// サーバー起動 (ポート8080)
	log.Println("Starting Athenetic API server on port 8080...")
	e.Logger.Fatal(e.Start(":8080"))
}