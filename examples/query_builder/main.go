package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"

	"github.com/pechorka/cruder/pkg/dbx"
)

// User represents a user in the database
type User struct {
	ID   int    `db:"id,auto"`
	Name string `db:"name"`
}

// InsertUserInput represents the input for inserting a user
type InsertUserInput struct {
	Name string `db:"name"`
}

// Repository represents a data repository
type Repository struct {
	db dbx.DB
}

// Pre-compiled query - this happens at package initialization
var insertUserQuery = dbx.Returning[InsertUserInput, User](dbx.Insert[InsertUserInput]("users")).Compile()

func (r *Repository) InsertUser(ctx context.Context, input InsertUserInput) (User, error) {
	q := insertUserQuery.New(input) // new knows what type to expect for input
	return q.ExecContext(ctx, r.db) // ExecContext knows what type to expect for returning
}

func main() {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	query, args := insertUserQuery.PreviewQuery(InsertUserInput{Name: "John"})
	fmt.Println(query)
	fmt.Println(args)

	repo := &Repository{db: db}

	user, err := repo.InsertUser(context.Background(), InsertUserInput{Name: "John"})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(user)
}
