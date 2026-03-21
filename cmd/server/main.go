package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/router"
	"backend-pretest-ai/pkg/ai"
	"backend-pretest-ai/pkg/storage"
)

func main() {
	// 1. Load semua env variable
	config.Load()

	// 2. Connect ke PostgreSQL
	config.ConnectDatabase()

	// 3. Init Cloudflare R2
	storage.InitR2()

	// 4. Init Genkit client
	ai.InitGenkit()

	// 5. Setup Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Backend-Pretest-AI",
	})

	// 6. Register semua route
	router.Setup(app)

	// 7. Jalankan server
	port := config.Cfg.App.Port
	log.Printf("[server] running on http://localhost:%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("[server] failed to start: %v", err)
	}
}
