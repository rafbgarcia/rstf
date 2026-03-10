package rstf

import (
	"database/sql"
	"fmt"
	"time"
)

// App holds application-level configuration initialized at startup.
// The layout's main.go exports an App(*rstf.App) function to configure it.
type App struct {
	db                    *sql.DB
	requestBodyLimitBytes int64
	maxConcurrentRequests int
	maxQueuedRequests     int
	queueTimeout          time.Duration
	readHeaderTimeout     time.Duration
	readTimeout           time.Duration
	writeTimeout          time.Duration
	idleTimeout           time.Duration
}

const (
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultWriteTimeout      = 30 * time.Second
	DefaultIdleTimeout       = 2 * time.Minute
)

// NewApp creates an unconfigured App.
func NewApp() *App {
	return &App{
		requestBodyLimitBytes: DefaultBodyLimit,
		maxConcurrentRequests: DefaultMaxConcurrentRequests,
		maxQueuedRequests:     DefaultMaxQueuedRequests,
		queueTimeout:          DefaultQueueTimeout,
		readHeaderTimeout:     DefaultReadHeaderTimeout,
		readTimeout:           DefaultReadTimeout,
		writeTimeout:          DefaultWriteTimeout,
		idleTimeout:           DefaultIdleTimeout,
	}
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

// SetRequestBodyLimitBytes sets the maximum request body size accepted by BindJSON.
func (a *App) SetRequestBodyLimitBytes(limit int64) error {
	if limit <= 0 {
		return fmt.Errorf("request body limit must be greater than zero bytes")
	}
	a.requestBodyLimitBytes = limit
	return nil
}

// RequestBodyLimitBytes returns the configured request body limit.
func (a *App) RequestBodyLimitBytes() int64 {
	if a.requestBodyLimitBytes <= 0 {
		return DefaultBodyLimit
	}
	return a.requestBodyLimitBytes
}

// SetMaxConcurrentRequests sets the maximum number of requests handled concurrently.
func (a *App) SetMaxConcurrentRequests(limit int) error {
	if limit <= 0 {
		return fmt.Errorf("max concurrent requests must be greater than zero")
	}
	a.maxConcurrentRequests = limit
	return nil
}

// MaxConcurrentRequests returns the configured concurrent request limit.
func (a *App) MaxConcurrentRequests() int {
	if a.maxConcurrentRequests <= 0 {
		return DefaultMaxConcurrentRequests
	}
	return a.maxConcurrentRequests
}

// SetMaxQueuedRequests sets the maximum number of queued requests.
func (a *App) SetMaxQueuedRequests(limit int) error {
	if limit <= 0 {
		return fmt.Errorf("max queued requests must be greater than zero")
	}
	a.maxQueuedRequests = limit
	return nil
}

// MaxQueuedRequests returns the configured queued request limit.
func (a *App) MaxQueuedRequests() int {
	if a.maxQueuedRequests <= 0 {
		return DefaultMaxQueuedRequests
	}
	return a.maxQueuedRequests
}

// SetQueueTimeout sets how long a request can wait in queue before returning overload.
func (a *App) SetQueueTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("queue timeout must be greater than zero")
	}
	a.queueTimeout = timeout
	return nil
}

// QueueTimeout returns the configured queue wait timeout.
func (a *App) QueueTimeout() time.Duration {
	if a.queueTimeout <= 0 {
		return DefaultQueueTimeout
	}
	return a.queueTimeout
}

// SetReadHeaderTimeout sets the maximum time allowed to read request headers.
func (a *App) SetReadHeaderTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("read header timeout must be greater than zero")
	}
	a.readHeaderTimeout = timeout
	return nil
}

// ReadHeaderTimeout returns the configured request header read timeout.
func (a *App) ReadHeaderTimeout() time.Duration {
	if a.readHeaderTimeout <= 0 {
		return DefaultReadHeaderTimeout
	}
	return a.readHeaderTimeout
}

// SetReadTimeout sets the maximum duration for reading the entire request.
func (a *App) SetReadTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("read timeout must be greater than zero")
	}
	a.readTimeout = timeout
	return nil
}

// ReadTimeout returns the configured full request read timeout.
func (a *App) ReadTimeout() time.Duration {
	if a.readTimeout <= 0 {
		return DefaultReadTimeout
	}
	return a.readTimeout
}

// SetWriteTimeout sets the maximum duration before timing out a response write.
func (a *App) SetWriteTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("write timeout must be greater than zero")
	}
	a.writeTimeout = timeout
	return nil
}

// WriteTimeout returns the configured response write timeout.
func (a *App) WriteTimeout() time.Duration {
	if a.writeTimeout <= 0 {
		return DefaultWriteTimeout
	}
	return a.writeTimeout
}

// SetIdleTimeout sets how long keep-alive connections may remain idle.
func (a *App) SetIdleTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("idle timeout must be greater than zero")
	}
	a.idleTimeout = timeout
	return nil
}

// IdleTimeout returns the configured keep-alive idle timeout.
func (a *App) IdleTimeout() time.Duration {
	if a.idleTimeout <= 0 {
		return DefaultIdleTimeout
	}
	return a.idleTimeout
}

// Close shuts down the application, closing the database connection pool if open.
func (a *App) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
