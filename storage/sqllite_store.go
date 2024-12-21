package storage

import "zombiezen.com/go/sqlite"

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