package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// RetrieveExample retrieves an example by ID and security context
func (r *ExampleRepository) RetrieveExample(
	ctx context.Context,
	exampleID int,
	securityContext int,
) (*model.ExampleModel, error) {
	// First check if record exists (including deleted ones)
	checkQuery := `
		SELECT delete_user_id
		FROM example
		WHERE example_id = $1 AND security_context_id = $2`

	var deleteUserID *int
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		return r.db.QueryRow(ctx, checkQuery,
			exampleID, securityContext).Scan(&deleteUserID)
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to check example existence: %w", err)
	}

	// If record exists but is deleted, return ErrDeleted
	if deleteUserID != nil {
		return nil, ErrDeleted
	}

	// Record exists and is not deleted, retrieve full data
	query := `
		SELECT example_id, example_text, example_int, example_float,
		       example_int_array, example_float_array, security_context_id,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM example
		WHERE example_id = $1 AND security_context_id = $2 AND delete_user_id IS NULL`

	var example model.ExampleModel
	err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		return r.db.QueryRow(ctx, query, exampleID, securityContext).Scan(
			&example.ExampleID, &example.ExampleText, &example.ExampleInt, &example.ExampleFloat,
			&example.ExampleIntArray, &example.ExampleFloatArray, &example.SecurityContext,
			&example.CreateTS, &example.CreateUserID, &example.UpdateTS, &example.UpdateUserID,
			&example.DeleteTS, &example.DeleteUserID,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get example: %w", err)
	}

	return &example, nil
}

// CreateExample creates a new example within a security context
func (r *ExampleRepository) CreateExample(
	ctx context.Context,
	example *model.ExampleModel, securityContext int,
) (*model.ExampleModel, error) {
	// Validate that the example can only be created in the user's security context
	if example.SecurityContext != securityContext {
		return nil, fmt.Errorf(
			"cannot create example in different security context",
		)
	}

	query := `
		INSERT INTO example (
			example_text, example_int, example_float, example_int_array,
			example_float_array, security_context_id, create_ts, create_user_id,
			update_ts, update_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING example_id`

	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		return r.db.QueryRow(ctx, query,
			example.ExampleText, example.ExampleInt, example.ExampleFloat,
			example.ExampleIntArray, example.ExampleFloatArray,
			example.SecurityContext, example.CreateTS, example.CreateUserID,
			example.UpdateTS, example.UpdateUserID,
		).Scan(&example.ExampleID)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create example: %w", err)
	}

	return example, nil
}

// UpdateExample updates an existing example with optimistic locking
func (r *ExampleRepository) UpdateExample(
	ctx context.Context,
	example *model.ExampleModel,
	originalUpdateTS float64,
) error {
	query := `
		UPDATE example
		SET example_text = $1, example_int = $2, example_float = $3,
		    example_int_array = $4, example_float_array = $5,
		    update_ts = $6, update_user_id = $7
		WHERE example_id = $8 AND security_context_id = $9 AND update_ts = $10
		      AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(ctx, query,
			example.ExampleText, example.ExampleInt, example.ExampleFloat,
			example.ExampleIntArray, example.ExampleFloatArray,
			example.UpdateTS, example.UpdateUserID,
			example.ExampleID, example.SecurityContext, originalUpdateTS,
		)
		return execErr
	})

	if err != nil {
		return fmt.Errorf("failed to update example: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Assess the state of the example
		var deleteUserID *int
		checkQuery := `SELECT delete_user_id FROM example
			WHERE example_id = $1 AND security_context_id = $2`
		err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
			return r.db.QueryRow(ctx, checkQuery,
				example.ExampleID, example.SecurityContext).Scan(&deleteUserID)
		})
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("failed to assess example state: %w", err)
		}

		if deleteUserID != nil {
			return ErrDeleted
		}
		return ErrConflict
	}

	return nil
}

// DeleteExample soft deletes an example with optimistic locking
func (r *ExampleRepository) DeleteExample(
	ctx context.Context,
	exampleID int,
	securityContext int,
	originalUpdateTS float64,
	deleteTS float64,
	deleteUserID int,
) (int, error) {
	query := `
		UPDATE example
		SET delete_ts = $1, delete_user_id = $2
		WHERE example_id = $3 AND security_context_id = $4 AND update_ts = $5
		      AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(
			ctx,
			query,
			deleteTS,
			deleteUserID,
			exampleID,
			securityContext,
			originalUpdateTS,
		)
		return execErr
	})

	if err != nil {
		return 0, fmt.Errorf("failed to delete example: %w", err)
	}

	if result.RowsAffected() == 0 {
		var deleteUserIDCheck *int
		checkQuery := `SELECT delete_user_id FROM example
			WHERE example_id = $1 AND security_context_id = $2`
		err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
			return r.db.QueryRow(ctx, checkQuery,
				exampleID, securityContext).Scan(&deleteUserIDCheck)
		})
		if err != nil {
			if err == pgx.ErrNoRows {
				return 0, ErrNotFound
			}
			return 0, fmt.Errorf("failed to assess example state: %w", err)
		}

		if deleteUserIDCheck != nil {
			return 0, ErrDeleted
		}
		return 0, ErrConflict
	}

	return exampleID, nil
}

// RetrieveExampleList retrieves multiple examples by IDs and security context,
// or all if idList is empty
// If afterUpdateTs > 0, only returns examples with update_ts > afterUpdateTs
func (r *ExampleRepository) RetrieveExampleList(
	ctx context.Context,
	idList []int,
	securityContext int,
	afterUpdateTs float64,
) ([]*model.ExampleModel, error) {
	selectClause := `
		SELECT example_id, example_text, example_int, example_float,
		       example_int_array, example_float_array, security_context_id,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM example`
	orderBy := ` ORDER BY example_id`

	// Scoping and the incremental-fetch rule come from buildScopedWhere.
	whereClause, argList := buildScopedWhere(
		securityContext, afterUpdateTs, "example_id", idList)

	query := selectClause + whereClause + orderBy //+ " Limit 1000"

	var rowList pgx.Rows
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var queryErr error
		rowList, queryErr = r.db.Query(ctx, query, argList...)
		return queryErr
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query examples: %w", err)
	}
	defer rowList.Close()

	exampleList := make([]*model.ExampleModel, 0)
	for rowList.Next() {
		var example model.ExampleModel
		err := rowList.Scan(
			&example.ExampleID,
			&example.ExampleText,
			&example.ExampleInt,
			&example.ExampleFloat,
			&example.ExampleIntArray,
			&example.ExampleFloatArray,
			&example.SecurityContext,
			&example.CreateTS,
			&example.CreateUserID,
			&example.UpdateTS,
			&example.UpdateUserID,
			&example.DeleteTS,
			&example.DeleteUserID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan example: %w", err)
		}
		exampleList = append(exampleList, &example)
	}

	if err := rowList.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return exampleList, nil
}

// CreateExampleList creates multiple examples in a transaction within a security context
func (r *ExampleRepository) CreateExampleList(
	ctx context.Context,
	exampleList []*model.ExampleModel, securityContext int,
) ([]*model.ExampleModel, error) {
	if len(exampleList) == 0 {
		return []*model.ExampleModel{}, nil
	}

	// Validate that all examples can only be created in the user's security context
	for _, example := range exampleList {
		if example.SecurityContext != securityContext {
			return nil, fmt.Errorf(
				"cannot create example in different security context",
			)
		}
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `
		INSERT INTO example (
			example_text, example_int, example_float, example_int_array,
			example_float_array, security_context_id, create_ts, create_user_id,
			update_ts, update_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING example_id`

	for i, example := range exampleList {
		err := tx.QueryRow(ctx, query,
			example.ExampleText, example.ExampleInt, example.ExampleFloat,
			example.ExampleIntArray, example.ExampleFloatArray,
			example.SecurityContext, example.CreateTS, example.CreateUserID,
			example.UpdateTS, example.UpdateUserID,
		).Scan(&exampleList[i].ExampleID)

		if err != nil {
			return nil, fmt.Errorf("failed to create example %d: %w", i, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return exampleList, nil
}

// UpdateExampleList updates multiple examples in a transaction within a security context
func (r *ExampleRepository) UpdateExampleList(
	ctx context.Context,
	exampleList []*model.ExampleModel,
	originalUpdateTSList []float64,
	securityContext int,
) error {
	if len(exampleList) == 0 {
		return nil
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `
		UPDATE example
		SET example_text = $1, example_int = $2, example_float = $3, example_int_array = $4, example_float_array = $5,
		    update_ts = $6, update_user_id = $7
		WHERE example_id = $8 AND security_context_id = $9 AND update_ts = $10 AND delete_user_id IS NULL`

	for i, example := range exampleList {
		result, err := tx.Exec(
			ctx,
			query,
			example.ExampleText,
			example.ExampleInt,
			example.ExampleFloat,
			example.ExampleIntArray,
			example.ExampleFloatArray,
			example.UpdateTS,
			example.UpdateUserID,
			example.ExampleID,
			securityContext,
			originalUpdateTSList[i],
		)
		if err != nil {
			return fmt.Errorf(
				"failed to update example %d: %w",
				example.ExampleID,
				err,
			)
		}
		if result.RowsAffected() != 1 {
			return ErrExpectationFailed
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteExampleList deletes multiple examples in a transaction
func (r *ExampleRepository) DeleteExampleList(
	ctx context.Context,
	idList []int,
	updateTimestampList []float64,
	securityContext int,
	deleteTS float64,
	deleteUserID int,
) ([]int, error) {
	if len(idList) == 0 {
		return []int{}, nil
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `
		UPDATE example
		SET delete_ts = $1, delete_user_id = $2
		WHERE example_id = $3 AND security_context_id = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	for i, id := range idList {
		_, err := tx.Exec(
			ctx,
			query,
			deleteTS,
			deleteUserID,
			id,
			securityContext,
			updateTimestampList[i],
		)
		if err != nil {
			return nil, fmt.Errorf("failed to delete example %d: %w", id, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return idList, nil
}
