package dao

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/sam-berry/ecfr-analyzer/server/data"
	"time"
)

type CfrStructureDAO struct {
	Db *sql.DB
}

// Insert inserts a new CFR structure element
func (d *CfrStructureDAO) Insert(
	ctx context.Context,
	structure *data.CfrStructure,
) error {
	id := uuid.New().String()

	_, err := d.Db.ExecContext(
		ctx,
		`INSERT INTO cfr_structure(
			structure_id, title_id, title_number, div_type, div_level,
			identifier, node_id, heading, text_content, word_count,
			parent_id, path, created_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		id,
		structure.TitleId,
		structure.TitleNumber,
		structure.DivType,
		structure.DivLevel,
		structure.Identifier,
		structure.NodeId,
		structure.Heading,
		structure.TextContent,
		structure.WordCount,
		structure.ParentId,
		structure.Path,
		time.Now().UTC(),
	)

	if err != nil {
		return fmt.Errorf("error inserting cfr structure: %w", err)
	}

	return nil
}

// BatchInsert inserts multiple CFR structure elements in a single transaction
func (d *CfrStructureDAO) BatchInsert(
	ctx context.Context,
	structures []*data.CfrStructure,
) error {
	if len(structures) == 0 {
		return nil
	}

	tx, err := d.Db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO cfr_structure(
			structure_id, title_id, title_number, div_type, div_level,
			identifier, node_id, heading, text_content, word_count,
			parent_id, path, created_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
	)
	if err != nil {
		return fmt.Errorf("error preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, structure := range structures {
		id := uuid.New().String()
		_, err := stmt.ExecContext(
			ctx,
			id,
			structure.TitleId,
			structure.TitleNumber,
			structure.DivType,
			structure.DivLevel,
			structure.Identifier,
			structure.NodeId,
			structure.Heading,
			structure.TextContent,
			structure.WordCount,
			structure.ParentId,
			structure.Path,
			time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("error inserting cfr structure: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// DeleteByTitleId deletes all structure elements for a given title
func (d *CfrStructureDAO) DeleteByTitleId(
	ctx context.Context,
	titleId int,
) error {
	_, err := d.Db.ExecContext(
		ctx,
		`DELETE FROM cfr_structure WHERE title_id = $1`,
		titleId,
	)

	if err != nil {
		return fmt.Errorf("error deleting cfr structures for title %d: %w", titleId, err)
	}

	return nil
}

// FindByTitleNumber finds all structure elements for a given title number
func (d *CfrStructureDAO) FindByTitleNumber(
	ctx context.Context,
	titleNumber int,
) ([]*data.CfrStructure, error) {
	rows, err := d.Db.QueryContext(
		ctx,
		`SELECT id, structure_id, title_id, title_number, div_type, div_level,
			identifier, node_id, heading, text_content, word_count,
			parent_id, path, created_timestamp
		FROM cfr_structure
		WHERE title_number = $1
		ORDER BY path`,
		titleNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("error finding cfr structures by title: %w", err)
	}
	defer rows.Close()

	return d.scanStructures(rows)
}

// FindByDivType finds all structure elements of a given type
func (d *CfrStructureDAO) FindByDivType(
	ctx context.Context,
	titleNumber int,
	divType string,
) ([]*data.CfrStructure, error) {
	rows, err := d.Db.QueryContext(
		ctx,
		`SELECT id, structure_id, title_id, title_number, div_type, div_level,
			identifier, node_id, heading, text_content, word_count,
			parent_id, path, created_timestamp
		FROM cfr_structure
		WHERE title_number = $1 AND div_type = $2
		ORDER BY path`,
		titleNumber,
		divType,
	)
	if err != nil {
		return nil, fmt.Errorf("error finding cfr structures by type: %w", err)
	}
	defer rows.Close()

	return d.scanStructures(rows)
}

// FindByPath finds a structure element by its hierarchical path
func (d *CfrStructureDAO) FindByPath(
	ctx context.Context,
	titleNumber int,
	path string,
) (*data.CfrStructure, error) {
	var structure data.CfrStructure
	err := d.Db.QueryRowContext(
		ctx,
		`SELECT id, structure_id, title_id, title_number, div_type, div_level,
			identifier, node_id, heading, text_content, word_count,
			parent_id, path, created_timestamp
		FROM cfr_structure
		WHERE title_number = $1 AND path = $2`,
		titleNumber,
		path,
	).Scan(
		&structure.InternalId,
		&structure.Id,
		&structure.TitleId,
		&structure.TitleNumber,
		&structure.DivType,
		&structure.DivLevel,
		&structure.Identifier,
		&structure.NodeId,
		&structure.Heading,
		&structure.TextContent,
		&structure.WordCount,
		&structure.ParentId,
		&structure.Path,
		&structure.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding cfr structure by path: %w", err)
	}

	return &structure, nil
}

// scanStructures scans multiple rows into CfrStructure slice
func (d *CfrStructureDAO) scanStructures(rows *sql.Rows) ([]*data.CfrStructure, error) {
	var structures []*data.CfrStructure

	for rows.Next() {
		var structure data.CfrStructure
		err := rows.Scan(
			&structure.InternalId,
			&structure.Id,
			&structure.TitleId,
			&structure.TitleNumber,
			&structure.DivType,
			&structure.DivLevel,
			&structure.Identifier,
			&structure.NodeId,
			&structure.Heading,
			&structure.TextContent,
			&structure.WordCount,
			&structure.ParentId,
			&structure.Path,
			&structure.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning cfr structure row: %w", err)
		}

		structures = append(structures, &structure)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cfr structure rows: %w", err)
	}

	return structures, nil
}
