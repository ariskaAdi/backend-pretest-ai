package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/firebase/genkit/go/genkit"
	"github.com/joho/godotenv"

	"backend-pretest-ai/genkit/flows"
)

func main() {
	// Load .env
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("[genkit] .env not found, using system env")
	}

	if os.Getenv("GROQ_API_KEY") == "" {
		log.Fatal("[genkit] GROQ_API_KEY tidak di-set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init Genkit — model calls handled directly via Groq API
	g := genkit.Init(ctx)

	// Register semua flows dan simpan referensinya
	summarizeFlow := flows.RegisterSummarizeFlow(g)
	generateQuizFlow := flows.RegisterGenerateQuizFlow(g)

	log.Println("[genkit] flows registered: summarizeModule, generateQuiz")

	port := os.Getenv("GENKIT_PORT")
	if port == "" {
		port = "3400"
	}

	// Mount setiap flow sebagai HTTP handler
	mux := http.NewServeMux()
	mux.HandleFunc("/summarizeModule", genkit.Handler(summarizeFlow))
	mux.HandleFunc("/generateQuiz", genkit.Handler(generateQuizFlow))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Jalankan server di goroutine
	go func() {
		log.Printf("[genkit] server running on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
