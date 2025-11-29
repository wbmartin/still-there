package repository

import (
	"context"
	"fmt"
	"strings"

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
func (r *ExampleRepository) GetExample(ctx context.Context, exampleID int, securityContext int) (*model.ExampleModel, error) {
	query := `
		SELECT example_id, example_text, example_int, example_float, example_int_array, example_float_array,
		       security_context, create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM example
		WHERE example_id = $1 AND security_context = $2 AND delete_user_id IS NULL`

	var example model.ExampleModel
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
func (r *ExampleRepository) CreateExample(ctx context.Context, example *model.ExampleModel) (*model.ExampleModel, error) {
	query := `
		INSERT INTO example (example_text, example_int, example_float, example_int_array, example_float_array,
		                 security_context, create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING example_id`

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
func (r *ExampleRepository) UpdateExample(ctx context.Context, example *model.ExampleModel, originalUpdateTS float64) error {
	query := `
		UPDATE example
		SET example_text = $1, example_int = $2, example_float = $3, example_int_array = $4, example_float_array = $5,
		    update_ts = $6, update_user_id = $7
		WHERE example_id = $8 AND security_context = $9 AND update_ts = $10 AND delete_user_id IS NULL`

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
		checkQuery := `SELECT EXISTS(SELECT 1 FROM example WHERE example_id = $1 AND security_context = $2 AND delete_user_id IS NULL)`
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
func (r *ExampleRepository) DeleteExample(ctx context.Context, exampleID int, securityContext int, originalUpdateTS float64, deleteTS float64, deleteUserID int) (int, error) {
	query := `
		UPDATE example
		SET delete_ts = $1, delete_user_id = $2
		WHERE example_id = $3 AND security_context = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	result, err := r.db.Exec(ctx, query, deleteTS, deleteUserID, exampleID, securityContext, originalUpdateTS)

	if err != nil {
		return 0, fmt.Errorf("failed to delete example: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if the example exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM example WHERE example_id = $1 AND security_context = $2 AND delete_user_id IS NULL)`
		err := r.db.QueryRow(ctx, checkQuery, exampleID, securityContext).Scan(&exists)
		if err != nil {
			return 0, fmt.Errorf("failed to check example existence: %w", err)
		}

		if exists {
			return 0, fmt.Errorf("conflict: example was updated by another user")
		}
		return 0, fmt.Errorf("gone: example no longer exists")
	}

	return exampleID, nil
}

// GetExampleList retrieves multiple examples by IDs and security context
func (r *ExampleRepository) GetExampleList(ctx context.Context, ids []int, securityContext int) ([]*model.ExampleModel, error) {
	if len(ids) == 0 {
		return []*model.ExampleModel{}, nil
	}

	// Build IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+1)
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(ids)] = securityContext

	query := fmt.Sprintf(`
		SELECT example_id, example_text, example_int, example_float, example_int_array, example_float_array,
		       security_context, create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM example
		WHERE example_id IN (%s) AND security_context = $%d AND delete_user_id IS NULL`,
		strings.Join(placeholders, ","), len(ids)+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query examples: %w", err)
	}
	defer rows.Close()

	var examples []*model.ExampleModel
	for rows.Next() {
		var example model.ExampleModel
		err := rows.Scan(
			&example.ExampleID, &example.ExampleText, &example.ExampleInt, &example.ExampleFloat,
			&example.ExampleIntArray, &example.ExampleFloatArray, &example.SecurityContext,
			&example.CreateTS, &example.CreateUserID, &example.UpdateTS, &example.UpdateUserID,
			&example.DeleteTS, &example.DeleteUserID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan example: %w", err)
		}
		examples = append(examples, &example)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return examples, nil
}

// CreateExampleList creates multiple examples in a transaction
func (r *ExampleRepository) CreateExampleList(ctx context.Context, examples []*model.ExampleModel) ([]*model.ExampleModel, error) {
	if len(examples) == 0 {
		return []*model.ExampleModel{}, nil
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO example (example_text, example_int, example_float, example_int_array, example_float_array,
		                 security_context, create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING example_id`

	for i, example := range examples {
		err := tx.QueryRow(ctx, query,
			example.ExampleText, example.ExampleInt, example.ExampleFloat, example.ExampleIntArray, example.ExampleFloatArray,
			example.SecurityContext, example.CreateTS, example.CreateUserID, example.UpdateTS, example.UpdateUserID,
		).Scan(&examples[i].ExampleID)

		if err != nil {
			return nil, fmt.Errorf("failed to create example %d: %w", i, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return examples, nil
}

// UpdateExampleList updates multiple examples in a transaction
func (r *ExampleRepository) UpdateExampleList(ctx context.Context, examples []*model.ExampleModel) error {
	if len(examples) == 0 {
		return nil
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE example
		SET example_text = $1, example_int = $2, example_float = $3, example_int_array = $4, example_float_array = $5,
		    update_ts = $6, update_user_id = $7
		WHERE example_id = $8 AND security_context = $9 AND delete_user_id IS NULL`

	for _, example := range examples {
		_, err := tx.Exec(ctx, query,
			example.ExampleText, example.ExampleInt, example.ExampleFloat, example.ExampleIntArray, example.ExampleFloatArray,
			example.UpdateTS, example.UpdateUserID, example.ExampleID, example.SecurityContext,
		)
		if err != nil {
			return fmt.Errorf("failed to update example %d: %w", example.ExampleID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteExampleList deletes multiple examples in a transaction
func (r *ExampleRepository) DeleteExampleList(ctx context.Context, ids []int, updateTs []float64, securityContext int, deleteTS float64, deleteUserID int) ([]int, error) {
	if len(ids) == 0 {
		return []int{}, nil
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE example
		SET delete_ts = $1, delete_user_id = $2
		WHERE example_id = $3 AND security_context = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	for i, id := range ids {
		_, err := tx.Exec(ctx, query, deleteTS, deleteUserID, id, securityContext, updateTs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to delete example %d: %w", id, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ids, nil
}
