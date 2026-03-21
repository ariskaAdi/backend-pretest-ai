package router

import "github.com/gofiber/fiber/v2"

func Setup(app *fiber.App) {
	api := app.Group("/api/v1")

	// health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// TODO: register handler routes di sini
}
