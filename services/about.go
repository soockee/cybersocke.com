package services

import "github.com/soockee/cybersocke.com/storage"

type AboutService struct {
	store storage.Storage
}

func NewAboutService(store storage.Storage) *AboutService {
	return &AboutService{
		store: store,
	}
}

func (s *AboutService) GetAbout() []byte {
	return s.store.GetAbout()
}
