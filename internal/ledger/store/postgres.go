//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package store

import (
	"context"
	"database/sql"
	"encoding/hex"

	// Register postgres driver.
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/insolar/insolar/instrumentation/inslogger"
)

type PostgresDB struct {
	backend       *sql.DB
	insert        *sql.Stmt
	update        *sql.Stmt
	delete        *sql.Stmt
	selectByKey   *sql.Stmt
	cursor        *sql.Stmt
	reverseCursor *sql.Stmt
}

func NewPostgresDB(connStr string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open postgres")
	}
	insert, err := db.Prepare(`INSERT INTO records(key, value, scope) VALUES($1, $2, $3);`)
	if err != nil {
		db.Close()
		return nil, errors.Wrapf(err, "failed to make prepared insert")
	}
	update, err := db.Prepare(`UPDATE records SET value = $2 WHERE key = $1 AND scope = $3;`)
	if err != nil {
		db.Close()
		return nil, errors.Wrapf(err, "failed to make prepared update")
	}
	delete, err := db.Prepare(`DELETE FROM records WHERE key = $1 AND scope = $2;`)
	if err != nil {
		db.Close()
		return nil, errors.Wrapf(err, "failed to make prepared delete")
	}
	selectByKey, err := db.Prepare(`SELECT value FROM records WHERE key = $1 AND scope = $2;`)
	if err != nil {
		db.Close()
		return nil, errors.Wrapf(err, "failed to make prepared selectByKey")
	}
	cursor, err := db.Prepare(`SELECT key, value FROM records WHERE scope = $1 AND key >= $2 ORDER BY key;`)
	if err != nil {
		db.Close()
		return nil, errors.Wrapf(err, "failed to make prepared cursor")
	}
	reverseCursor, err := db.Prepare(`SELECT key, value FROM records WHERE scope = $1 AND key <= $2 ORDER BY key DESC;`)
	if err != nil {
		db.Close()
		return nil, errors.Wrapf(err, "failed to make prepared reversed cursor")
	}
	return &PostgresDB{
		backend:       db,
		insert:        insert,
		update:        update,
		delete:        delete,
		selectByKey:   selectByKey,
		cursor:        cursor,
		reverseCursor: reverseCursor,
	}, nil
}

func (p *PostgresDB) Stop(ctx context.Context) error {
	logger := inslogger.FromContext(ctx)
	defer logger.Info("PostgresDB: database closed")

	return p.backend.Close()
}

func (p *PostgresDB) Get(key Key) ([]byte, error) {
	hexKey := hex.EncodeToString(key.ID())
	var hexValue string
	err := p.selectByKey.QueryRow(hexKey, key.Scope()).Scan(&hexValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return []byte{}, errors.Wrapf(err, "failed to query value by key and scope")
	}

	value, err := hex.DecodeString(hexValue)
	if err != nil {
		return []byte{}, errors.New("failed to decode value from hex string")
	}
	return value, nil
}

func (p *PostgresDB) Set(key Key, value []byte) error {
	hexKey := hex.EncodeToString(key.ID())
	hexValue := hex.EncodeToString(value)
	_, err := p.Get(key)
	if err == nil {
		result, err := p.update.Exec(hexKey, hexValue, key.Scope())
		return checkErr(result, err, "failed to update row in db")

	}
	result, err := p.insert.Exec(hexKey, hexValue, key.Scope())
	return checkErr(result, err, "failed to insert row in db")
}

func (p *PostgresDB) Delete(key Key) error {
	hexKey := hex.EncodeToString(key.ID())
	result, err := p.delete.Exec(hexKey, key.Scope())
	return checkErr(result, err, "failed to delete row in db")
}

func checkErr(result sql.Result, err error, msg string) error {
	if err != nil {
		return errors.Wrapf(err, msg)
	}
	if result != nil {
		if rows, supportErr := result.RowsAffected(); supportErr == nil && rows != 1 {
			return errors.Wrapf(err, msg)
		}
	}
	return nil
}

func (p *PostgresDB) NewIterator(pivot Key, reverse bool) Iterator {
	var (
		err  error
		rows *sql.Rows
	)
	hexPivot := hex.EncodeToString(pivot.ID())
	if reverse {
		rows, err = p.reverseCursor.Query(pivot.Scope(), hexPivot)
		if err != nil {
			return nil
		}
	} else {
		rows, err = p.cursor.Query(pivot.Scope(), hexPivot)
		if err != nil {
			return nil
		}
	}
	return &postgresIterator{cursor: rows}
}

type postgresIterator struct {
	cursor *sql.Rows
}

func (pi *postgresIterator) Close() {
	if pi == nil {
		return
	}
	logger := inslogger.FromContext(context.Background())
	err := pi.cursor.Close()
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to close postgres iterator"))
	}
}

func (pi *postgresIterator) Next() bool {
	if pi == nil {
		return false
	}
	return pi.cursor.Next()
}

func (pi *postgresIterator) Key() []byte {
	var hexKey, hexValue string
	err := pi.cursor.Scan(&hexKey, &hexValue)
	if err != nil {
		return []byte{}
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return []byte{}
	}
	return key
}

func (pi *postgresIterator) Value() []byte {
	var hexKey, hexValue string
	err := pi.cursor.Scan(&hexKey, &hexValue)
	if err != nil {
		return []byte{}
	}
	value, err := hex.DecodeString(hexValue)
	if err != nil {
		return []byte{}
	}
	return value
}
