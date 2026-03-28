package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	"backend-pretest-ai/pkg/response"
)

type LynkHandler struct {
	lynkService service.LynkService
}

func NewLynkHandler(lynkService service.LynkService) *LynkHandler {
	return &LynkHandler{
		lynkService: lynkService,
	}
}

// POST /api/v1/webhook/lynk
// HandleLynkWebhook godoc
// @Summary      Handle Lynk.id webhook
// @Description  Receive payment notifications from Lynk.id and update user quotas
// @Tags         webhook
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      400  {object}  response.APIResponse
// @Router       /webhook/lynk [post]
func (h *LynkHandler) HandleWebhook(c *fiber.Ctx) error {
	// 1. Secret Validation
	// Based on issue.md recommendation: check X-Webhook-Secret header
	secret := c.Get("X-Webhook-Secret")
	if secret != config.Cfg.App.LynkWebhookSecret {
		log.Printf("[lynk_handler] unauthorized webhook attempt from IP: %s", c.IP())
		return response.Unauthorized(c, "invalid webhook secret")
	}

	var req dto.LynkWebhookPayload
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[lynk_handler] failed to parse body: %v", err)
		return response.BadRequest(c, "format request tidak valid")
	}

	// 2. Process Webhook via Service
	if err := h.lynkService.ProcessWebhook(c.Context(), req); err != nil {
		if errors.Is(err, service.ErrTransactionAlreadyProcessed) {
			// Jika sudah diproses, return 200 OK agar Lynk tidak retry redundan
			return response.OK(c, "transaksi sudah diproses sebelumnya", nil)
		}
		
		log.Printf("[lynk_handler] error processing webhook: %v", err)
		// Tetap return 200 jika ini error business logic yang tidak butuh retry
		// Tapi di sini kita return 200 sesuai best practice webhook agar tidak stuck retry
		return response.OK(c, "webhook received with internal warnings", nil)
	}

	return response.OK(c, "webhook processed successfully", nil)
}
