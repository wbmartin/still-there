package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/model"
)

// ExampleRepository handles database operations for examples
type ExampleRepository struct {
	db *pgxpool.Pool
}

// NewExampleRepository creates a new example repository
func NewExampleRepository(db *pgxpool.Pool) *ExampleRepository {
	return &ExampleRepository{db: db}
}

// GetExample retrieves an example by ID and security context
func (r *ExampleRepository) GetExample(ctx context.Context, exampleID int, securityContext int) (*model.Example, error) {
	query := `
		SELECT note_id, note_text, note_int, note_float, note_int_array, note_float_array,
		       security_context, create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM example
		WHERE note_id = $1 AND security_context = $2 AND delete_user_id IS NULL`

	var example model.Example
	err := r.db.QueryRow(ctx, query, exampleID, securityContext).Scan(
		&example.ExampleID, &example.ExampleText, &example.ExampleInt, &example.ExampleFloat,
		&example.ExampleIntArray, &example.ExampleFloatArray, &example.SecurityContext,
		&example.CreateTS, &example.CreateUserID, &example.UpdateTS, &example.UpdateUserID,
		&example.DeleteTS, &example.DeleteUserID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("example not found")
		}
		return nil, fmt.Errorf("failed to get example: %w", err)
	}

	return &example, nil
}

// CreateExample creates a new example
func (r *ExampleRepository) CreateExample(ctx context.Context, example *model.Example) (*model.Example, error) {
	query := `
		INSERT INTO example (note_text, note_int, note_float, note_int_array, note_float_array,
		                 security_context, create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING note_id`

	err := r.db.QueryRow(ctx, query,
		example.ExampleText, example.ExampleInt, example.ExampleFloat, example.ExampleIntArray, example.ExampleFloatArray,
		example.SecurityContext, example.CreateTS, example.CreateUserID, example.UpdateTS, example.UpdateUserID,
	).Scan(&example.ExampleID)

	if err != nil {
		return nil, fmt.Errorf("failed to create example: %w", err)
	}

	return example, nil
}

// UpdateExample updates an existing example with optimistic locking
func (r *ExampleRepository) UpdateExample(ctx context.Context, example *model.Example, originalUpdateTS float64) error {
	query := `
		UPDATE example
		SET note_text = $1, note_int = $2, note_float = $3, note_int_array = $4, note_float_array = $5,
		    update_ts = $6, update_user_id = $7
		WHERE note_id = $8 AND security_context = $9 AND update_ts = $10 AND delete_user_id IS NULL`

	result, err := r.db.Exec(ctx, query,
		example.ExampleText, example.ExampleInt, example.ExampleFloat, example.ExampleIntArray, example.ExampleFloatArray,
		example.UpdateTS, example.UpdateUserID, example.ExampleID, example.SecurityContext, originalUpdateTS,
	)

	if err != nil {
		return fmt.Errorf("failed to update example: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if the example exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM example WHERE note_id = $1 AND security_context = $2 AND delete_user_id IS NULL)`
		err := r.db.QueryRow(ctx, checkQuery, example.ExampleID, example.SecurityContext).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check example existence: %w", err)
		}

		if exists {
			return fmt.Errorf("conflict: example was updated by another user")
		}
		return fmt.Errorf("gone: example no longer exists")
	}

	return nil
}

// DeleteExample soft deletes an example with optimistic locking
func (r *ExampleRepository) DeleteExample(ctx context.Context, exampleID int, securityContext int, originalUpdateTS float64, deleteTS float64, deleteUserID int) error {
	query := `
		UPDATE example
		SET delete_ts = $1, delete_user_id = $2
		WHERE note_id = $3 AND security_context = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	result, err := r.db.Exec(ctx, query, deleteTS, deleteUserID, exampleID, securityContext, originalUpdateTS)

	if err != nil {
		return fmt.Errorf("failed to delete example: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if the example exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM example WHERE note_id = $1 AND security_context = $2 AND delete_user_id IS NULL)`
		err := r.db.QueryRow(ctx, checkQuery, exampleID, securityContext).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check example existence: %w", err)
		}

		if exists {
			return fmt.Errorf("conflict: example was updated by another user")
		}
		return fmt.Errorf("gone: example no longer exists")
	}

	return nil
}
