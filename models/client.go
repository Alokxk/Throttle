package models

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"time"
)

type Client struct {
	ID               string    `json:"client_id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	APIKey           string    `json:"api_key,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	IsActive         bool      `json:"is_active"`
	DefaultAlgorithm string    `json:"default_algorithm"`
}

func CreateClient(ctx context.Context, db *sql.DB, name, email, apiKey, keyPrefix, keyHash, defaultAlgorithm string) (*Client, error) {
	client := &Client{}

	query := `
		INSERT INTO clients (name, email, api_key, key_prefix, api_key_hash, default_algorithm)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, email, created_at, is_active, default_algorithm
	`

	err := db.QueryRowContext(ctx, query, name, email, apiKey, keyPrefix, keyHash, defaultAlgorithm).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.CreatedAt,
		&client.IsActive,
		&client.DefaultAlgorithm,
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func GetClientByAPIKey(ctx context.Context, db *sql.DB, apiKey string) (*Client, error) {
	if len(apiKey) < 8 {
		return nil, sql.ErrNoRows
	}

	keyPrefix := apiKey[:8]

	rows, err := db.QueryContext(ctx, `
		SELECT id, name, email, api_key, COALESCE(api_key_hash, ''), created_at, is_active, default_algorithm
		FROM clients
		WHERE (key_prefix = $1 OR ((key_prefix IS NULL OR key_prefix = '') AND api_key = $2))
		AND is_active = true
	`, keyPrefix, apiKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		client := &Client{}
		var storedKey, storedHash string

		err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Email,
			&storedKey,
			&storedHash,
			&client.CreatedAt,
			&client.IsActive,
			&client.DefaultAlgorithm,
		)
		if err != nil {
			return nil, err
		}

		if storedHash != "" {
			computed := sha256.Sum256([]byte(apiKey))
			computedHex := hex.EncodeToString(computed[:])
			if subtle.ConstantTimeCompare([]byte(storedHash), []byte(computedHex)) == 1 {
				client.APIKey = storedKey
				return client, nil
			}
		} else {
			if storedKey == apiKey {
				client.APIKey = storedKey
				return client, nil
			}
		}
	}

	return nil, sql.ErrNoRows
}
