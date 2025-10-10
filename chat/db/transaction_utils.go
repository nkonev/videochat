package db

import (
	"context"
	"fmt"
)

// Performs txFunc transactionally with the result.
// Deliberately doesn't support "nesting" to keep the code simple - to have only one place with TransactWithResult().
func TransactWithResult[T any](ctx context.Context, db *DB, txFunc func(*Tx) (T, error)) (ret T, err error) {
	tx, err := db.Begin(ctx, db.lgr)
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.SafeRollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			tx.SafeRollback() // err is non-nil; don't change it
			err = fmt.Errorf("Rolled back: %w", err)
		} else {
			err = tx.Commit() // err is nil; if Commit returns error update err
		}
	}()
	ret, err = txFunc(tx)
	return ret, err
}

// Performs txFunc transactionally without any result.
// Deliberately doesn't support "nesting" to keep the code simple - to have only one place with Transact().
func Transact(ctx context.Context, db *DB, txFunc func(*Tx) error) (err error) {
	_, err = TransactWithResult(ctx, db, func(tx *Tx) (any, error) {
		return nil, txFunc(tx)
	})
	return err
}
