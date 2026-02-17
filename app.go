package rstf

import "database/sql"

// App holds application-level configuration initialized at startup.
// The layout's main.go exports an App(*rstf.App) function to configure it.
type App struct {
	db *sql.DB
}

// NewApp creates an unconfigured App.
func NewApp() *App {
	return &App{}
}

// Database opens a connection pool using the given driver and DSN.
func (a *App) Database(driverName, dataSourceName string) error {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}
	a.db = db
	return nil
}

// DB returns the configured *sql.DB, or nil if no database was configured.
func (a *App) DB() *sql.DB {
	return a.db
}

// Close shuts down the application, closing the database connection pool if open.
func (a *App) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
