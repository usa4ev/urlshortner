package database

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/usa4ev/urlshortner/internal/storage/storageerrors"

	_ "github.com/golang/mock/mockgen/model"
	_ "github.com/jackc/pgx/stdlib"
)

type (
	database struct {
		*sql.DB
		ctx    context.Context
		stmnts statements
		buffer *asyncBuf
	}
	statements struct {
		storeURL     *sql.Stmt
		storeSession *sql.Stmt
	}
	deletedItems []Item
	Item         struct {
		ID      string
		url     string
		UserID  string
		deleted bool
	}
	asyncBuf struct {
		*bufio.Writer
		mx sync.Mutex
	}
)

func New(dsn string, ctx context.Context) database {
	var (
		db  database
		err error
	)
	db.ctx = ctx

	db.DB, err = sql.Open("pgx", dsn)
	if err != nil {
		panic(err.Error())
	}

	err = db.initDB()
	if err != nil {
		panic(err.Error())
	}

	db.stmnts, err = db.prepareStatements()
	if err != nil {
		panic(err.Error())
	}

	db.initBuffer()

	return db
}

func (db database) initDB() error {
	query := `CREATE TABLE IF NOT EXISTS users (
				id VARCHAR(100) PRIMARY KEY,
					session VARCHAR(256));`

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error %s when creating table users", err)
		return err
	}

	query = `CREATE TABLE IF NOT EXISTS urls (
					url VARCHAR(100) NOT NULL UNIQUE,
					id VARCHAR(100) PRIMARY KEY UNIQUE,
					user_id VARCHAR(38),
					deleted BOOLEAN,
					FOREIGN KEY (user_id)
				REFERENCES users (id));`

	rows, err = db.Query(query)
	if err != nil {
		return err
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error %s when creating table urls", err)
		return err
	}

	return err
}

func (db *database) initBuffer() {
	db.buffer = &asyncBuf{Writer: bufio.NewWriter(db)}
}

func (db database) prepareStatements() (statements, error) {
	storeURL, err := db.PrepareContext(db.ctx, "INSERT INTO urls(id, url, user_id, deleted) VALUES ($1, $2, $3, FALSE) ON CONFLICT (url) DO NOTHING")
	if err != nil {
		return statements{}, err
	}

	storeSession, err := db.PrepareContext(db.ctx, "INSERT INTO users(id, session) VALUES ($1, $2)")
	if err != nil {
		return statements{}, err
	}

	return statements{storeURL, storeSession}, nil
}

func (db database) StoreURL(id, url, userid string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ctx, cancelfunc := context.WithTimeout(db.ctx, 5*time.Second)
	defer cancelfunc()

	txStmt := tx.StmtContext(ctx, db.stmnts.storeURL)

	res, err := txStmt.ExecContext(ctx, id, url, userid)
	if err != nil {
		log.Printf("Error %s when inserting row into users table", err)
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}

	if rows == 0 {
		return storageerrors.ErrConflict
	}

	return tx.Commit()
}

func (db database) LoadURL(id string) (string, error) {
	var (
		url, query string
		deleted    bool
		rows       *sql.Rows
		err        error
	)

	ctx, cancelfunc := context.WithTimeout(db.ctx, 5*time.Second)
	defer cancelfunc()

	query = "SELECT url, deleted FROM urls WHERE id = $1"
	rows, err = db.QueryContext(ctx, query, id)
	if err != nil {
		log.Printf("Error %s when lodaing URL using id %v", err, id)
		return "", err
	}

	defer rows.Close()

	if err = rows.Err(); err != nil {
		log.Printf("Error %s when lodaing URL using id %v", err, id)
		return "", err
	}

	if !rows.Next() {
		return "", nil
	}
	err = rows.Scan(&url, &deleted)
	if err != nil {
		log.Printf("Error %s when scanning query results; id %v", err, id)
		return "", err
	}

	if deleted {
		return "", storageerrors.ErrURLGone
	}

	return url, nil
}

func (db database) LoadUrlsByUser(add func(id, url string), userid string) error {
	var (
		id, url, query string
		rows           *sql.Rows
		err            error
	)

	ctx, cancelfunc := context.WithTimeout(db.ctx, 5*time.Second)
	defer cancelfunc()

	query = "SELECT id, url FROM urls WHERE user_id = $1"
	rows, err = db.QueryContext(ctx, query, userid)
	if err != nil {
		log.Printf("Error %s when lodaing URL using id %v", err, userid)
		return err
	}

	if rows.Err() != nil {
		return rows.Err()
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&id, &url)
		if err != nil {
			log.Printf("Error %s when scanning query results; id %v", err, userid)
			return err
		}

		add(id, url)
	}

	return nil
}

func (db database) LoadUser(session string) (string, error) {
	var url string
	query := "SELECT id FROM users WHERE session = $1"

	ctx, cancelfunc := context.WithTimeout(db.ctx, 5*time.Second)
	defer cancelfunc()

	rows, err := db.QueryContext(ctx, query, session)
	if err != nil {
		log.Printf("Error %s when lodaing session using id %v", err, session)
		return "", err
	}

	if rows.Err() != nil {
		return "", rows.Err()
	}

	defer rows.Close()

	if !rows.Next() {
		return "", nil
	}

	err = rows.Scan(&url)
	if err != nil {
		log.Printf("Error %s when scanning query results; id %v", err, session)
		return "", err
	}

	return url, err
}

func (db database) StoreSession(id, session string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ctx, cancelfunc := context.WithTimeout(db.ctx, 5*time.Second)
	defer cancelfunc()

	txStmt := tx.StmtContext(ctx, db.stmnts.storeSession)

	res, err := txStmt.ExecContext(ctx, id, session)
	if err != nil {
		log.Printf("Error %s when inserting row into users table", err)
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}

	return tx.Commit()
}

func (db database) DeleteURLs(userID string, ids []string) error {
	its := make(deletedItems, len(ids))

	for i, id := range ids {
		its[i] = Item{ID: id, UserID: userID}
	}

	b, err := itsToBytes(its)
	if err != nil {
		return fmt.Errorf("failed to represent struct slice as bytes: %w\n", err)
	}

	db.buffer.mx.Lock()
	defer db.buffer.mx.Unlock()
	_, err = db.buffer.Write(b)

	if err != nil {
		return fmt.Errorf("failed to represent write to buf: %w\n", err)
	}

	return nil
}

func itsToBytes(its deletedItems) ([]byte, error) {
	w := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(w)

	err := enc.Encode(its)
	if err != nil {
		return nil, fmt.Errorf("enconig failed: %w\n", err)
	}

	return w.Bytes(), nil
}

func (db database) Write(p []byte) (int, error) {
	var err error

	pr, pw := io.Pipe()
	defer pr.Close()

	errCh := make(chan error)

	go func() {
		defer pw.Close()

		n, err := pw.Write(p)
		if err != nil {
			errCh <- fmt.Errorf("failed to pipe: %w\n", err)
			return
		}
		//close(errCh)
		fmt.Printf("written %v bytes\n", n)
	}()

	dec := gob.NewDecoder(pr)
	res := make(deletedItems, 0)
	var d deletedItems

loop:
	for {
		select {
		case err = <-errCh:
			if err != nil {
				return 0, err
			}
		default:
			if err = dec.Decode(&d); errors.Is(err, io.EOF) {
				break loop
			} else if err != nil {
				return 0, fmt.Errorf("failed to decode struct %w\n", err)
			}
			fmt.Printf("decoded struct %v\n", d)
			res = append(res, d...)
		}
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("deconig failed: %w\n", err)
	}

	fmt.Printf("deconig successful")

	valueStrings := make([]string, 0, len(res))
	valueArgs := make([]interface{}, 0, len(res)*2)

	c := 1
	for _, v := range res {
		valueStrings = append(valueStrings, fmt.Sprintf("($%v, $%v, TRUE)", c, c+1))
		valueArgs = append(valueArgs, v.ID, v.UserID)
		c += 2
	}

	stmt := fmt.Sprintf("UPDATE urls SET deleted = tmp.deleted from (values %s) as tmp (id, user_id, deleted) "+
		"WHERE urls.user_id = tmp.user_id AND urls.id = tmp.id AND urls.deleted = false",
		strings.Join(valueStrings, ","))

	rows, err := db.Exec(stmt, valueArgs...)

	if err != nil {
		return 0, fmt.Errorf("failed to update buffered urls in database: %w\n", err)
	}

	_, err = rows.RowsAffected()

	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w\n", err)
	}

	return len(p), nil
}

func Pingdb(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	err = conn.PingContext(ctx)
	if err != nil {
		return err
	}

	if err = db.PingContext(ctx); err != nil {
		return err
	}

	return err
}

func (db database) Flush() error {
	return db.buffer.Flush()
}
