package db

import (
	"context"
	"github.com/stretchr/testify/assert"
	"nkonev.name/chat/config"
	. "nkonev.name/chat/logger"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	shutdown()
	os.Exit(retCode)
}

func shutdown() {
	if dbInstance != nil {
		dbInstance.Close()
	}
}

var dbInstance *DB
var lgr = NewLogger()

func setup() {
	config.InitViper()

	d, err := ConfigureDb(lgr, nil)
	dbInstance = d

	if err != nil {
		lgr.Panicf("Error during getting db connection for test: %v", err)
	} else {
		d.RecreateDb()
	}
}

func TestTransactionPositive(t *testing.T) {
	err := Transact(context.Background(), dbInstance, func(tx *Tx) error {
		if _, err := tx.Exec("CREATE TABLE t1(a text UNIQUE)"); err != nil {
			return err
		}
		if _, err := tx.Exec("insert into t1(a) VALUES ('lorem')"); err != nil {
			return err
		}
		return nil
	})
	assert.Nil(t, err)

	row := dbInstance.QueryRow("SELECT a FROM t1")
	var a string
	err = row.Scan(&a)
	assert.Nil(t, err)
	assert.Equal(t, "lorem", a)
}

func TestTransactionNegative(t *testing.T) {
	_, err := dbInstance.Exec("CREATE TABLE t2(a text UNIQUE)")
	assert.Nil(t, err)

	err = Transact(context.Background(), dbInstance, func(tx *Tx) error {
		if _, err := tx.Exec("insert into t2(a) VALUES ('lorem')"); err != nil {
			return err
		}
		if _, err := tx.Exec("insert into t2(a) VALUES ('lorem')"); err != nil {
			return err
		}
		return nil
	})
	assert.NotNil(t, err)

	row := dbInstance.QueryRow("SELECT a FROM t2")
	var a string
	err = row.Scan(&a)
	assert.NotNil(t, err)
	s := err.Error()
	assert.Equal(t, `sql: no rows in result set`, s)
}

func TestTransactionWithResultPositive(t *testing.T) {
	id, err := TransactWithResult(context.Background(), dbInstance, func(tx *Tx) (int64, error) {
		if _, err := tx.Exec("CREATE TABLE tr1(id BIGSERIAL PRIMARY KEY, a text UNIQUE)"); err != nil {
			return 0, err
		}
		res := tx.QueryRow(`INSERT INTO tr1(a) VALUES ('lorem') RETURNING id`)
		var id int64
		if err := res.Scan(&id); err != nil {
			lgr.Errorf("Error during getting chat id %v", err)
			return 0, err
		}

		return id, nil
	})
	assert.Nil(t, err)

	assert.True(t, id != 0)

	row := dbInstance.QueryRow("SELECT a FROM tr1 WHERE id = $1", id)
	var a string
	err = row.Scan(&a)
	assert.Nil(t, err)
	assert.Equal(t, "lorem", a)
}

func TestTransactionWithResultNegative(t *testing.T) {
	_, err := dbInstance.Exec("CREATE TABLE tr2(id BIGSERIAL PRIMARY KEY, a text UNIQUE)")
	assert.Nil(t, err)

	idRaw, err := TransactWithResult(context.Background(), dbInstance, func(tx *Tx) (int64, error) {
		res := tx.QueryRow(`INSERT INTO tr2(a) VALUES ('lorem') RETURNING id`)
		var id int64
		if err := res.Scan(&id); err != nil {
			lgr.Errorf("Error during getting chat id %v", err)
			return 0, err
		}
		if _, err := tx.Exec("insert into tr2(a) VALUES ('lorem')"); err != nil {
			return 0, err
		}

		return id, nil
	})
	assert.NotNil(t, err)
	assert.Equal(t, int64(0), idRaw)

	row := dbInstance.QueryRow("SELECT a FROM tr2")
	var a string
	err = row.Scan(&a)
	assert.NotNil(t, err)
	s := err.Error()
	assert.Equal(t, `sql: no rows in result set`, s)
}
