package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	_ "github.com/golang/mock/mockgen/model"
	_ "github.com/jackc/pgx/stdlib"
)

type (
	database struct {
		*sql.DB
		ctx    context.Context
		stmnts statements
	}
	statements struct {
		storeURL     *sql.Stmt
		storeSession *sql.Stmt
	}
)

var ErrConflict = errors.New("URL has already been shortened")

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
		(err.Error())
	}

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

func (db database) prepareStatements() (statements, error) {
	storeURL, err := db.PrepareContext(db.ctx, "INSERT INTO urls(id, url, user_id) VALUES ($1, $2, $3) ON CONFLICT (url) DO NOTHING")
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
		return ErrConflict
	}

	return tx.Commit()
}

func (db database) LoadURL(id string) (string, error) {
	var (
		url, query string
		rows       *sql.Rows
		err        error
	)

	ctx, cancelfunc := context.WithTimeout(db.ctx, 5*time.Second)
	defer cancelfunc()

	query = "SELECT url FROM urls WHERE id = $1"
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
	err = rows.Scan(&url)
	if err != nil {
		log.Printf("Error %s when scanning query results; id %v", err, id)
		return "", err
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
	return nil
}
