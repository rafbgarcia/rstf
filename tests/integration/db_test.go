package integration_test

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestApp creates an App with an in-memory SQLite database and a seeded posts table.
func setupTestApp(t *testing.T) *rstf.App {
	t.Helper()
	app := rstf.NewApp()
	require.NoError(t, app.Database("sqlite3", ":memory:"))
	_, err := app.DB().Exec(`
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT FALSE
		)`)
	require.NoError(t, err)
	_, err = app.DB().Exec(`
		INSERT INTO posts (title, published) VALUES
			('First Post', true),
			('Draft Post', false)`)
	require.NoError(t, err)
	t.Cleanup(func() { app.Close() })
	return app
}

func TestApp_Database(t *testing.T) {
	app := rstf.NewApp()
	require.NoError(t, app.Database("sqlite3", ":memory:"))
	defer app.Close()

	require.NotNil(t, app.DB())
	require.NoError(t, app.DB().Ping(), "ping failed")
}

func TestApp_Database_InvalidDriver(t *testing.T) {
	app := rstf.NewApp()
	err := app.Database("nonexistent", "foo")
	require.Error(t, err)
}

func TestApp_DB_NilWhenNotConfigured(t *testing.T) {
	app := rstf.NewApp()
	require.Nil(t, app.DB())
}

func TestContext_DB_RawSQL(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	ctx := rstf.NewContext(req)
	ctx.DB = app.DB()

	rows, err := ctx.DB.QueryContext(ctx.Request.Context(),
		"SELECT title, published FROM posts WHERE published = ?", true)
	require.NoError(t, err)
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var title string
		var published bool
		require.NoError(t, rows.Scan(&title, &published))
		titles = append(titles, title)
	}

	assert.Equal(t, []string{"First Post"}, titles)
}

func TestContext_DB_GORM(t *testing.T) {
	app := setupTestApp(t)

	// User wraps the framework's *sql.DB with GORM.
	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: app.DB(),
	}), &gorm.Config{})
	require.NoError(t, err)

	type Post struct {
		ID        int
		Title     string
		Published bool
	}

	var posts []Post
	require.NoError(t, gormDB.Where("published = ?", true).Find(&posts).Error)
	require.Len(t, posts, 1)
	assert.Equal(t, "First Post", posts[0].Title)
}

func TestContext_DB_GORM_Create(t *testing.T) {
	app := setupTestApp(t)

	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: app.DB(),
	}), &gorm.Config{})
	require.NoError(t, err)

	type Post struct {
		ID        int
		Title     string
		Published bool
	}

	newPost := Post{Title: "New Post", Published: true}
	require.NoError(t, gormDB.Create(&newPost).Error)
	assert.NotZero(t, newPost.ID)

	// Verify via raw SQL that it was actually inserted.
	var count int
	app.DB().QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'New Post'").Scan(&count)
	assert.Equal(t, 1, count)
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
	require.NoError(t, db.Select(&posts, "SELECT * FROM posts WHERE published = ?", true))
	require.Len(t, posts, 1)
	assert.Equal(t, "First Post", posts[0].Title)
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
	require.NoError(t, err)
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		require.NoError(t, rows.StructScan(&p))
		posts = append(posts, p)
	}

	require.Len(t, posts, 1)
	assert.Equal(t, "First Post", posts[0].Title)
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
	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "First Post", posts[0].Title)
}

// TestContext_DB_Transaction verifies that transactions via *sql.DB work correctly.
func TestContext_DB_Transaction(t *testing.T) {
	app := setupTestApp(t)

	tx, err := app.DB().Begin()
	require.NoError(t, err)

	_, err = tx.Exec("INSERT INTO posts (title, published) VALUES (?, ?)", "TX Post", true)
	if err != nil {
		_ = tx.Rollback()
	}
	require.NoError(t, err)

	// Before commit, verify it's visible inside the tx but not outside.
	var countInTx int
	tx.QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'TX Post'").Scan(&countInTx)
	assert.Equal(t, 1, countInTx)

	var countOutside int
	app.DB().QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'TX Post'").Scan(&countOutside)
	assert.Equal(t, 0, countOutside)

	require.NoError(t, tx.Commit())

	// After commit, visible everywhere.
	app.DB().QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'TX Post'").Scan(&countOutside)
	assert.Equal(t, 1, countOutside)
}

// TestContext_DB_Rollback verifies rollback discards changes.
func TestContext_DB_Rollback(t *testing.T) {
	app := setupTestApp(t)

	tx, err := app.DB().Begin()
	require.NoError(t, err)

	tx.Exec("INSERT INTO posts (title, published) VALUES (?, ?)", "Rolled Back", true)
	tx.Rollback()

	var count int
	app.DB().QueryRow("SELECT COUNT(*) FROM posts WHERE title = 'Rolled Back'").Scan(&count)
	assert.Equal(t, 0, count)
}
