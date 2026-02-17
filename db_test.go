package rstf_test

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"testing"

	rstf "github.com/rafbgarcia/rstf"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestApp creates an App with an in-memory SQLite database and a seeded posts table.
func setupTestApp(t *testing.T) *rstf.App {
	t.Helper()
	app := rstf.NewApp()
	if err := app.Database("sqlite3", ":memory:"); err != nil {
		t.Fatal(err)
	}
	_, err := app.DB().Exec(`
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT FALSE
		)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB().Exec(`
		INSERT INTO posts (title, published) VALUES
			('First Post', true),
			('Draft Post', false)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Close() })
	return app
}

func TestApp_Database(t *testing.T) {
	app := rstf.NewApp()
	if err := app.Database("sqlite3", ":memory:"); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	if app.DB() == nil {
		t.Fatal("expected non-nil *sql.DB")
	}
	if err := app.DB().Ping(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestApp_Database_InvalidDriver(t *testing.T) {
	app := rstf.NewApp()
	err := app.Database("nonexistent", "foo")
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
}

func TestApp_DB_NilWhenNotConfigured(t *testing.T) {
	app := rstf.NewApp()
	if app.DB() != nil {
		t.Fatal("expected nil DB when not configured")
	}
}

func TestContext_DB_RawSQL(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	ctx := rstf.NewContext(req)
	ctx.DB = app.DB()

	rows, err := ctx.DB.QueryContext(ctx.Request.Context(),
		"SELECT title, published FROM posts WHERE published = ?", true)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var title string
		var published bool
		if err := rows.Scan(&title, &published); err != nil {
			t.Fatal(err)
		}
		titles = append(titles, title)
	}

	if len(titles) != 1 || titles[0] != "First Post" {
		t.Errorf("expected [First Post], got %v", titles)
	}
}

func TestContext_DB_GORM(t *testing.T) {
	app := setupTestApp(t)

	// User wraps the framework's *sql.DB with GORM.
	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: app.DB(),
	}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	type Post struct {
		ID        int
		Title     string
		Published bool
	}

	var posts []Post
	if err := gormDB.Where("published = ?", true).Find(&posts).Error; err != nil {
		t.Fatal(err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 published post, got %d", len(posts))
	}
	if posts[0].Title != "First Post" {
		t.Errorf("expected 'First Post', got %q", posts[0].Title)
	}
}

func TestContext_DB_GORM_Create(t *testing.T) {
	app := setupTestApp(t)

	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: app.DB(),
	}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	type Post struct {
		ID        int
		Title     string
		Published bool
	}

	newPost := Post{Title: "New Post", Published: true}
	if err := gormDB.Create(&newPost).Error; err != nil {
		t.Fatal(err)
	}
	if newPost.ID == 0 {
		t.Error("expected auto-generated ID")
	}

	// Verify via raw SQL that it was actually inserted.
	var count int
	app.DB().QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'New Post'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row with title 'New Post', got %d", count)
	}
}

func TestContext_DB_Sqlx(t *testing.T) {
	app := setupTestApp(t)

	// User wraps the framework's *sql.DB with sqlx.
	db := sqlx.NewDb(app.DB(), "sqlite3")

	type Post struct {
		ID        int    `db:"id"`
		Title     string `db:"title"`
		Published bool   `db:"published"`
	}

	var posts []Post
	if err := db.Select(&posts, "SELECT * FROM posts WHERE published = ?", true); err != nil {
		t.Fatal(err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 published post, got %d", len(posts))
	}
	if posts[0].Title != "First Post" {
		t.Errorf("expected 'First Post', got %q", posts[0].Title)
	}
}

func TestContext_DB_Sqlx_NamedQuery(t *testing.T) {
	app := setupTestApp(t)
	db := sqlx.NewDb(app.DB(), "sqlite3")

	type Post struct {
		ID        int    `db:"id"`
		Title     string `db:"title"`
		Published bool   `db:"published"`
	}

	rows, err := db.NamedQuery("SELECT * FROM posts WHERE published = :published",
		map[string]any{"published": true})
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.StructScan(&p); err != nil {
			t.Fatal(err)
		}
		posts = append(posts, p)
	}

	if len(posts) != 1 || posts[0].Title != "First Post" {
		t.Errorf("expected [First Post], got %v", posts)
	}
}

// TestContext_DB_SqlcPattern tests the pattern sqlc generates:
// standalone functions/methods that take context.Context and *sql.DB.
func TestContext_DB_SqlcPattern(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := rstf.NewContext(req)
	ctx.DB = app.DB()

	// sqlc generates a Queries struct wrapping *sql.DB:
	//   type Queries struct { db *sql.DB }
	//   func New(db *sql.DB) *Queries { return &Queries{db: db} }
	//   func (q *Queries) ListPublishedPosts(ctx context.Context) ([]Post, error) { ... }
	//
	// Simulate that pattern:
	type Post struct {
		ID        int
		Title     string
		Published bool
	}

	listPublishedPosts := func(db *sql.DB, reqCtx context.Context) ([]Post, error) {
		rows, err := db.QueryContext(reqCtx,
			"SELECT id, title, published FROM posts WHERE published = ?", true)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var posts []Post
		for rows.Next() {
			var p Post
			if err := rows.Scan(&p.ID, &p.Title, &p.Published); err != nil {
				return nil, err
			}
			posts = append(posts, p)
		}
		return posts, nil
	}

	posts, err := listPublishedPosts(ctx.DB, ctx.Request.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(posts) != 1 || posts[0].Title != "First Post" {
		t.Errorf("expected 1 published post 'First Post', got %v", posts)
	}
}

// TestContext_DB_Transaction verifies that transactions via *sql.DB work correctly.
func TestContext_DB_Transaction(t *testing.T) {
	app := setupTestApp(t)

	tx, err := app.DB().Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Exec("INSERT INTO posts (title, published) VALUES (?, ?)", "TX Post", true)
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}

	// Before commit, verify it's visible inside the tx but not outside.
	var countInTx int
	tx.QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'TX Post'").Scan(&countInTx)
	if countInTx != 1 {
		t.Errorf("expected 1 inside tx, got %d", countInTx)
	}

	var countOutside int
	app.DB().QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'TX Post'").Scan(&countOutside)
	if countOutside != 0 {
		t.Errorf("expected 0 outside tx before commit, got %d", countOutside)
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// After commit, visible everywhere.
	app.DB().QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'TX Post'").Scan(&countOutside)
	if countOutside != 1 {
		t.Errorf("expected 1 after commit, got %d", countOutside)
	}
}

// TestContext_DB_Rollback verifies rollback discards changes.
func TestContext_DB_Rollback(t *testing.T) {
	app := setupTestApp(t)

	tx, err := app.DB().Begin()
	if err != nil {
		t.Fatal(err)
	}

	tx.Exec("INSERT INTO posts (title, published) VALUES (?, ?)", "Rolled Back", true)
	tx.Rollback()

	var count int
	app.DB().QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'Rolled Back'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 after rollback, got %d", count)
	}
}
