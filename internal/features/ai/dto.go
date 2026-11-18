package ai

// GenerateRequest adalah payload POST /api/posts/ai-generate.
type GenerateRequest struct {
	Topic            string `json:"topic" binding:"required,min=3,max=255"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}

// Source adalah satu sumber rujukan hasil penelusuran Search Engine.
type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// GenerateResponse adalah draf artikel hasil AI beserta sumber rujukannya.
type GenerateResponse struct {
	Title         string   `json:"title"`
	Category      string   `json:"category"`
	Excerpt       string   `json:"excerpt"`
	Content       string   `json:"content"`
	Sources       []Source `json:"sources"`
	UsedWebSearch bool     `json:"used_web_search"`
}

// Message adalah satu pesan Chat Completions.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
