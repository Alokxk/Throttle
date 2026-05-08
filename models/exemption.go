package models

import (
	"database/sql"
	"time"
)

type Exemption struct {
	ID         string    `json:"id"`
	ClientID   string    `json:"client_id"`
	Identifier string    `json:"identifier"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

func CreateExemption(db *sql.DB, clientID, identifier, reason string) (*Exemption, error) {
	exemption := &Exemption{}

	query := `
		INSERT INTO exemptions (client_id, identifier, reason)
		VALUES ($1, $2, $3)
		RETURNING id, client_id, identifier, reason, created_at
	`

	err := db.QueryRow(query, clientID, identifier, reason).Scan(
		&exemption.ID,
		&exemption.ClientID,
		&exemption.Identifier,
		&exemption.Reason,
		&exemption.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return exemption, nil
}

func IsExempted(db *sql.DB, clientID, identifier string) (bool, error) {
	var id string
	err := db.QueryRow(`
		SELECT id FROM exemptions
		WHERE client_id = $1 AND identifier = $2
	`, clientID, identifier).Scan(&id)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func DeleteExemption(db *sql.DB, clientID, identifier string) error {
	var id string
	err := db.QueryRow(`
		DELETE FROM exemptions
		WHERE client_id = $1 AND identifier = $2
		RETURNING id
	`, clientID, identifier).Scan(&id)

	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	return err
}

func ListExemptions(db *sql.DB, clientID string) ([]*Exemption, error) {
	rows, err := db.Query(`
		SELECT id, client_id, identifier, reason, created_at
		FROM exemptions
		WHERE client_id = $1
		ORDER BY created_at ASC
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exemptions []*Exemption
	for rows.Next() {
		e := &Exemption{}
		err := rows.Scan(&e.ID, &e.ClientID, &e.Identifier, &e.Reason, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		exemptions = append(exemptions, e)
	}

	return exemptions, nil
}
