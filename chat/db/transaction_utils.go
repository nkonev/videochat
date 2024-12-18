package db

import "context"

func TransactWithResult[T any](ctx context.Context, db *DB, txFunc func(*Tx) (T, error)) (ret T, err error) {
	tx, err := db.Begin(ctx, db.lgr)
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback() // err is non-nil; don't change it
		} else {
			err = tx.Commit() // err is nil; if Commit returns error update err
		}
	}()
	ret, err = txFunc(tx)
	return ret, err
}

func Transact(ctx context.Context, db *DB, txFunc func(*Tx) error) (err error) {
	tx, err := db.Begin(ctx, db.lgr)
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback() // err is non-nil; don't change it
		} else {
			err = tx.Commit() // err is nil; if Commit returns error update err
		}
	}()
	err = txFunc(tx)
	return err
}
