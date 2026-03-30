package middleware

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// InitLogger — setup logrus, buat folder log kalau belum ada
func InitLogger() {
	Logger = logrus.New()

	// Buat folder log di root project kalau belum ada
	logDir := "log"
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		logrus.Fatalf("[logger] gagal membuat folder log: %v", err)
	}

	// Nama file log per hari: log/2026-03-21.log
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logrus.Fatalf("[logger] gagal membuka file log: %v", err)
	}

	Logger.SetOutput(file)
	Logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Hanya catat Warn ke atas (Warn, Error, Fatal, Panic)
	Logger.SetLevel(logrus.WarnLevel)

	logrus.Infof("[logger] initialized, writing to %s", logFile)
}

// LoggerMiddleware — tangkap request dengan response 4xx dan 5xx
func LoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Lanjutkan ke handler berikutnya
		err := c.Next()

		status := c.Response().StatusCode()
		duration := time.Since(start)

		fields := logrus.Fields{
			"method":   c.Method(),
			"path":     c.Path(),
			"status":   status,
			"duration": duration.String(),
			"ip":       c.IP(),
		}

		if errMsg, ok := c.Locals("responseError").(string); ok && errMsg != "" {
			fields["error"] = errMsg
		}

		// Log kalau status 4xx atau 5xx
		if status >= 500 {
			Logger.WithFields(fields).WithError(err).Error("server error")
		} else if status >= 400 {
			Logger.WithFields(fields).WithError(err).Warn("client error")
		}

		return err
	}
}
