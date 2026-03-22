package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/router"
	"backend-pretest-ai/pkg/ai"
	"backend-pretest-ai/pkg/mailer"
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

	// 5. Init Mailer
	mailer.InitMailer()

	// 6. Setup Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Backend Pretest AI API",
		// Matikan error handler default agar semua error lewat response helper
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		},
	})

	// 7. Register semua route
	router.Setup(app)

	// 8. Graceful shutdown — tangkap sinyal OS
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Jalankan server di goroutine agar tidak block
	go func() {
		port := config.Cfg.App.Port
		log.Printf("[server] running on http://localhost:%s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("[server] failed to start: %v", err)
		}
	}()

	// Block sampai sinyal diterima (Ctrl+C atau kill)
	<-quit
	log.Println("[server] shutting down gracefully...")

	// Shutdown Fiber — tunggu request yang sedang berjalan selesai
	if err := app.Shutdown(); err != nil {
		log.Printf("[server] error during shutdown: %v", err)
	}

	// Tutup koneksi DB
	if sqlDB, err := config.DB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("[database] error closing connection: %v", err)
		} else {
			log.Println("[database] connection closed")
		}
	}

	log.Println("[server] shutdown complete")
}

