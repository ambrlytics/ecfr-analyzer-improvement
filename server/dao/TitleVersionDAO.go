package dao

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/sam-berry/ecfr-analyzer/server/data"
	"time"
)

type TitleVersionDAO struct {
	Db *sql.DB
}

// Insert inserts a new title version
func (d *TitleVersionDAO) Insert(
	ctx context.Context,
	titleId int,
	titleNumber int,
	versionDate time.Time,
	content []byte,
) error {
	id := uuid.New().String()

	_, err := d.Db.ExecContext(
		ctx,
		`INSERT INTO title_version(
			version_id, title_id, title_number, content, version_date, created_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (title_number, version_date) DO UPDATE
		SET content = $4, created_timestamp = $6
		WHERE title_version.title_number = $3 AND title_version.version_date = $5`,
		id,
		titleId,
		titleNumber,
		string(content),
		versionDate,
		time.Now().UTC(),
	)

	if err != nil {
		return fmt.Errorf("error inserting title version: %w", err)
	}

	return nil
}

// FindByTitleNumber finds all versions for a given title number
func (d *TitleVersionDAO) FindByTitleNumber(
	ctx context.Context,
	titleNumber int,
) ([]*data.TitleVersion, error) {
	rows, err := d.Db.QueryContext(
		ctx,
		`SELECT id, version_id, title_id, title_number, version_date, created_timestamp
		FROM title_version
		WHERE title_number = $1
		ORDER BY version_date DESC`,
		titleNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("error finding title versions: %w", err)
	}
	defer rows.Close()

	var versions []*data.TitleVersion
	for rows.Next() {
		var version data.TitleVersion
		err := rows.Scan(
			&version.InternalId,
			&version.Id,
			&version.TitleId,
			&version.TitleNumber,
			&version.VersionDate,
			&version.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning title version row: %w", err)
		}

		versions = append(versions, &version)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating title version rows: %w", err)
	}

	return versions, nil
}

// FindByDate finds all title versions for a specific date
func (d *TitleVersionDAO) FindByDate(
	ctx context.Context,
	versionDate time.Time,
) ([]*data.TitleVersion, error) {
	rows, err := d.Db.QueryContext(
		ctx,
		`SELECT id, version_id, title_id, title_number, version_date, created_timestamp
		FROM title_version
		WHERE version_date = $1
		ORDER BY title_number`,
		versionDate,
	)
	if err != nil {
		return nil, fmt.Errorf("error finding title versions by date: %w", err)
	}
	defer rows.Close()

	var versions []*data.TitleVersion
	for rows.Next() {
		var version data.TitleVersion
		err := rows.Scan(
			&version.InternalId,
			&version.Id,
			&version.TitleId,
			&version.TitleNumber,
			&version.VersionDate,
			&version.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning title version row: %w", err)
		}

		versions = append(versions, &version)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating title version rows: %w", err)
	}

	return versions, nil
}

// FindByTitleAndDateRange finds versions for a title within a date range
func (d *TitleVersionDAO) FindByTitleAndDateRange(
	ctx context.Context,
	titleNumber int,
	startDate time.Time,
	endDate time.Time,
) ([]*data.TitleVersion, error) {
	rows, err := d.Db.QueryContext(
		ctx,
		`SELECT id, version_id, title_id, title_number, version_date, created_timestamp
		FROM title_version
		WHERE title_number = $1 AND version_date BETWEEN $2 AND $3
		ORDER BY version_date DESC`,
		titleNumber,
		startDate,
		endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("error finding title versions by date range: %w", err)
	}
	defer rows.Close()

	var versions []*data.TitleVersion
	for rows.Next() {
		var version data.TitleVersion
		err := rows.Scan(
			&version.InternalId,
			&version.Id,
			&version.TitleId,
			&version.TitleNumber,
			&version.VersionDate,
			&version.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning title version row: %w", err)
		}

		versions = append(versions, &version)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating title version rows: %w", err)
	}

	return versions, nil
}

// GetContentByVersion retrieves the XML content for a specific version
func (d *TitleVersionDAO) GetContentByVersion(
	ctx context.Context,
	titleNumber int,
	versionDate time.Time,
) (*data.TitleVersionWithContent, error) {
	var version data.TitleVersionWithContent
	var content string

	err := d.Db.QueryRowContext(
		ctx,
		`SELECT id, version_id, title_id, title_number, version_date, created_timestamp, content
		FROM title_version
		WHERE title_number = $1 AND version_date = $2`,
		titleNumber,
		versionDate,
	).Scan(
		&version.InternalId,
		&version.Id,
		&version.TitleId,
		&version.TitleNumber,
		&version.VersionDate,
		&version.CreatedAt,
		&content,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding title version with content: %w", err)
	}

	version.Content = content
	return &version, nil
}
