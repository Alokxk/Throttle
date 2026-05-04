package models

import (
	"database/sql"
	"time"
)

type Rule struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	Algorithm string    `json:"algorithm"`
	Limit     int       `json:"limit"`
	Window    int       `json:"window"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateRule(db *sql.DB, clientID, name, algorithm string, limit, window int) (*Rule, error) {
	rule := &Rule{}

	query := `
		INSERT INTO rules (client_id, name, algorithm, limit_val, window_seconds)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, client_id, name, algorithm, limit_val, window_seconds, created_at
	`

	err := db.QueryRow(query, clientID, name, algorithm, limit, window).Scan(
		&rule.ID,
		&rule.ClientID,
		&rule.Name,
		&rule.Algorithm,
		&rule.Limit,
		&rule.Window,
		&rule.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return rule, nil
}

func GetRuleByName(db *sql.DB, clientID, name string) (*Rule, error) {
	rule := &Rule{}

	query := `
		SELECT id, client_id, name, algorithm, limit_val, window_seconds, created_at
		FROM rules
		WHERE client_id = $1 AND name = $2
	`

	err := db.QueryRow(query, clientID, name).Scan(
		&rule.ID,
		&rule.ClientID,
		&rule.Name,
		&rule.Algorithm,
		&rule.Limit,
		&rule.Window,
		&rule.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return rule, nil
}

func ListRules(db *sql.DB, clientID string) ([]*Rule, error) {
	query := `
		SELECT id, client_id, name, algorithm, limit_val, window_seconds, created_at
		FROM rules
		WHERE client_id = $1
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*Rule
	for rows.Next() {
		rule := &Rule{}
		err := rows.Scan(
			&rule.ID,
			&rule.ClientID,
			&rule.Name,
			&rule.Algorithm,
			&rule.Limit,
			&rule.Window,
			&rule.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}
