package router

import (
	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/internal/handler"
	"backend-pretest-ai/internal/middleware"
	"backend-pretest-ai/internal/repository"
	"backend-pretest-ai/internal/service"
	"backend-pretest-ai/pkg/ai"
	pdfpkg "backend-pretest-ai/pkg/pdf"
	"backend-pretest-ai/pkg/storage"
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
	// Wire PDF vision extractor — fallback untuk PDF berbasis gambar/scan
	pdfpkg.VisionExtractor = ai.Client.VisionExtract

	// --- Wire dependencies ---
	userRepo := repository.NewUserRepository()
	userSvc := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userSvc)

	moduleRepo := repository.NewModuleRepository()
	moduleSvc := service.NewModuleService(moduleRepo, userRepo, storage.R2, ai.Client)
	moduleHandler := handler.NewModuleHandler(moduleSvc)

	summarySvc := service.NewSummaryService(moduleRepo)
	summaryHandler := handler.NewSummaryHandler(summarySvc)

	quizRepo := repository.NewQuizRepository()
	quizSvc := service.NewQuizService(quizRepo, moduleRepo, userRepo, ai.Client)
	quizHandler := handler.NewQuizHandler(quizSvc)

	lynkRepo := repository.NewLynkRepository()
	lynkSvc := service.NewLynkService(lynkRepo, userRepo)
	lynkHandler := handler.NewLynkHandler(lynkSvc)

	reviewRepo := repository.NewReviewRepository()
	reviewSvc := service.NewReviewService(reviewRepo, userRepo)
	reviewHandler := handler.NewReviewHandler(reviewSvc)

	api := app.Group("/api/v1")

	// Global middleware
	api.Use(middleware.LoggerMiddleware())

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// --- Webhook routes (public) ---
	webhook := api.Group("/webhook")
	webhook.Post("/lynk", lynkHandler.HandleWebhook)

	// --- Auth routes (public) ---
	auth := api.Group("/auth")
	auth.Post("/register", userHandler.Register)
	auth.Post("/verify-otp", userHandler.VerifyOTP)
	auth.Post("/resend-otp", userHandler.ResendOTP)
	auth.Post("/login", userHandler.Login)
	auth.Post("/logout", middleware.Auth(), userHandler.Logout)

	// --- User routes (protected) ---
	user := api.Group("/user", middleware.Auth())
	user.Get("/me", userHandler.GetMe)
	user.Post("/email/request-update", userHandler.RequestUpdateEmail)
	user.Post("/email/verify-update", userHandler.VerifyUpdateEmail)

	// --- Module routes (protected) ---
	modules := api.Group("/modules", middleware.Auth())
	modules.Post("/", moduleHandler.Upload)
	modules.Get("/", moduleHandler.GetAll)
	modules.Get("/:id", moduleHandler.GetByID)
	modules.Post("/:id/retry-summarize", moduleHandler.RetrySummarize)
	modules.Delete("/:id", moduleHandler.Delete)

	// --- Summary routes (protected) ---
	summary := api.Group("/summary", middleware.Auth())
	summary.Get("/:moduleId", summaryHandler.GetByModuleID)
	summary.Put("/:moduleId", summaryHandler.UpdateManual)

	// --- Quiz routes (protected) ---
	quiz := api.Group("/quiz", middleware.Auth())
	quiz.Post("/", quizHandler.Generate)
	quiz.Delete("/:id", quizHandler.Cancel)
	quiz.Post("/:id/submit", quizHandler.Submit)
	quiz.Post("/:id/explain", quizHandler.Explain)
	quiz.Post("/:id/retry", quizHandler.Retry)
	quiz.Get("/history", quizHandler.GetHistory)
	quiz.Get("/history/module/:moduleId", quizHandler.GetHistoryByModule)
	quiz.Get("/:id/result", quizHandler.GetResult)

	// --- Review routes ---
	api.Get("/reviews", reviewHandler.GetAll)
	reviews := api.Group("/reviews", middleware.Auth())
	reviews.Post("/", reviewHandler.Create)
	reviews.Put("/:id", reviewHandler.Update)
	reviews.Delete("/:id", reviewHandler.Delete)
}
