package dto

// --- Request ---

type UploadModuleRequest struct {
	Title string `form:"title" validate:"required,min=3,max=255"`
}

// --- Response ---

type ModuleResponse struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	FileURL      string `json:"file_url"`
	IsSummarized bool   `json:"is_summarized"`
	CreatedAt    string `json:"created_at"`
}

type ModuleDetailResponse struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	FileURL      string `json:"file_url"`
	Summary      string `json:"summary"`
	IsSummarized bool   `json:"is_summarized"`
	CreatedAt    string `json:"created_at"`
}
