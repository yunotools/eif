package dto

type CaptchaResponse struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}
