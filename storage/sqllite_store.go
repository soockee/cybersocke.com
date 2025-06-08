package storage

import (
	"errors"

	"zombiezen.com/go/sqlite"
)

type SQLiteStore struct {
	db *sqlite.Conn
}

func NewSQLiteStore() (*SQLiteStore, error) {
	conn, err := sqlite.OpenConn(":memory:", sqlite.OpenReadWrite)
	if err != nil {
		return nil, err
	}

	return &SQLiteStore{
		db: conn,
	}, nil
}

func (s *SQLiteStore) GetPost(id string) []byte {
	// placeholder
	return []byte("not implemented")
}

func (s *SQLiteStore) CreatePost(post Post) error {
	return errors.New("method not implemented")
}
