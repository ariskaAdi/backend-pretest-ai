package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/joho/godotenv"

	"backend-pretest-ai/genkit/flows"
)

func main() {
	// Load .env
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("[genkit] .env not found, using system env")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("[genkit] GEMINI_API_KEY tidak di-set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init Genkit dengan plugin Google AI (Gemini)
	g, err := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
	)
	if err != nil {
		log.Fatalf("[genkit] gagal init: %v", err)
	}

	// Register semua flows
	flows.RegisterSummarizeFlow(g)
	flows.RegisterGenerateQuizFlow(g)

	log.Println("[genkit] flows registered: summarizeModule, generateQuiz")

	port := os.Getenv("GENKIT_PORT")
	if port == "" {
		port = "3400"
	}

	// Jalankan server di goroutine
	srv := &http.Server{Addr: ":" + port}
	go func() {
		log.Printf("[genkit] server running on http://localhost:%s", port)
		if err := genkit.StartFlowServer(ctx, g, &genkit.FlowServerOptions{
			Addr: ":" + port,
		}); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[genkit] server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[genkit] shutting down gracefully...")
	cancel()
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("[genkit] error during shutdown: %v", err)
	}
	log.Println("[genkit] shutdown complete")
}
