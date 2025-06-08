package services

import (
	"context"
	"net/http"

	"github.com/gorilla/csrf"
)

type CSFRService struct {
}

func NewCSFRService(ctx context.Context) *CSFRService {
	return &CSFRService{}
}

func (s *CSFRService) Inject(w http.ResponseWriter, r *http.Request) {
	token := csrf.Token(r)
	w.Header().Set("X-CSRF-Token", token)
}
