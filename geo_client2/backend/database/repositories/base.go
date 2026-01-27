package repositories

import "database/sql"

// BaseRepository provides common database access.
type BaseRepository struct {
	db *sql.DB
}

// NewBaseRepository creates a new base repository.
func NewBaseRepository(db *sql.DB) *BaseRepository {
	return &BaseRepository{db: db}
}
