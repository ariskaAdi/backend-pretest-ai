package dto

// --- Request ---

type UpdateSummaryRequest struct {
	Summary string `json:"summary" validate:"required,min=10"`
}

// --- Response ---

type SummaryResponse struct {
	ModuleID     string `json:"module_id"`
	ModuleTitle  string `json:"module_title"`
	Summary      string `json:"summary"`
	IsSummarized    bool   `json:"is_summarized"`
	SummarizeFailed bool   `json:"summarize_failed"`
	UpdatedAt       string `json:"updated_at"`
}
