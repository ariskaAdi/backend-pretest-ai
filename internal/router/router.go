package router

import (
	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/internal/handler"
	"backend-pretest-ai/internal/middleware"
	"backend-pretest-ai/internal/repository"
	"backend-pretest-ai/internal/service"
)

// @title           Backend Pretest AI API
// @version         1.0
// @description     API Documentation for Backend Pretest AI project
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer " followed by your JWT token.
func Setup(app *fiber.App) {
	// --- Wire dependencies ---
	userRepo := repository.NewUserRepository()
	userSvc := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userSvc)

	moduleRepo := repository.NewModuleRepository()
	moduleSvc := service.NewModuleService(moduleRepo)
	moduleHandler := handler.NewModuleHandler(moduleSvc)

	summarySvc := service.NewSummaryService(moduleRepo)
	summaryHandler := handler.NewSummaryHandler(summarySvc)

	quizRepo := repository.NewQuizRepository()
	quizSvc := service.NewQuizService(quizRepo, moduleRepo)
	quizHandler := handler.NewQuizHandler(quizSvc)

	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// --- Auth routes (public) ---
	auth := api.Group("/auth")
	auth.Post("/register", userHandler.Register)
	auth.Post("/verify-otp", userHandler.VerifyOTP)
	auth.Post("/login", userHandler.Login)
	auth.Post("/logout", middleware.Auth(), userHandler.Logout)

	// --- User routes (protected) ---
	user := api.Group("/user", middleware.Auth())
	user.Post("/email/request-update", userHandler.RequestUpdateEmail)
	user.Post("/email/verify-update", userHandler.VerifyUpdateEmail)

	// --- Module routes (protected) ---
	modules := api.Group("/modules", middleware.Auth())
	modules.Post("/", moduleHandler.Upload)
	modules.Get("/", moduleHandler.GetAll)
	modules.Get("/:id", moduleHandler.GetByID)
	modules.Delete("/:id", moduleHandler.Delete)

	// --- Summary routes (protected) ---
	summary := api.Group("/summary", middleware.Auth())
	summary.Get("/:moduleId", summaryHandler.GetByModuleID)
	summary.Put("/:moduleId", summaryHandler.UpdateManual)

	// --- Quiz routes (protected) ---
	quiz := api.Group("/quiz", middleware.Auth())
	quiz.Post("/", quizHandler.Generate)
	quiz.Post("/:id/submit", quizHandler.Submit)
	quiz.Post("/:id/retry", quizHandler.Retry)
	quiz.Get("/history", quizHandler.GetHistory)
	quiz.Get("/history/module/:moduleId", quizHandler.GetHistoryByModule)
	quiz.Get("/:id/result", quizHandler.GetResult)
}
