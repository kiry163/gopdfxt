package gopdfxt

type Result struct {
	Articles    []Article         `json:"articles"`
	FailedPages []PageError       `json:"failed_pages,omitempty"`
	Details     ProcessingDetails `json:"details"`
}

type Article struct {
	Title   string    `json:"title"`
	Content string    `json:"content"`
	Pages   PageRange `json:"pages"`
}

type PageRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type PageError struct {
	PageIndex int    `json:"page_index"`
	Error     string `json:"error"`
}

type ProcessingDetails struct {
	PageCount      int `json:"page_count"`
	SucceededPages int `json:"succeeded_pages"`
	FailedPages    int `json:"failed_pages"`
	ModelCalls     int `json:"model_calls"`
	Retries        int `json:"retries"`
}
