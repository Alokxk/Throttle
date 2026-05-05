package models

import (
	"database/sql"
	"time"
)

type Client struct {
	ID               string    `json:"client_id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	APIKey           string    `json:"api_key"`
	CreatedAt        time.Time `json:"created_at"`
	IsActive         bool      `json:"is_active"`
	DefaultAlgorithm string    `json:"default_algorithm"`
}

func CreateClient(db *sql.DB, name, email, apiKey, defaultAlgorithm string) (*Client, error) {
	client := &Client{}

	query := `
		INSERT INTO clients (name, email, api_key, default_algorithm)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, email, api_key, created_at, is_active, default_algorithm
	`

	err := db.QueryRow(query, name, email, apiKey, defaultAlgorithm).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.APIKey,
		&client.CreatedAt,
		&client.IsActive,
		&client.DefaultAlgorithm,
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func GetClientByAPIKey(db *sql.DB, apiKey string) (*Client, error) {
	client := &Client{}

	query := `
		SELECT id, name, email, api_key, created_at, is_active, default_algorithm
		FROM clients
		WHERE api_key = $1 AND is_active = true
	`

	err := db.QueryRow(query, apiKey).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.APIKey,
		&client.CreatedAt,
		&client.IsActive,
		&client.DefaultAlgorithm,
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
