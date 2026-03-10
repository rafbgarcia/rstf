package rstf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAppRequestBodyLimit_Default(t *testing.T) {
	app := NewApp()
	require.Equal(t, DefaultBodyLimit, app.RequestBodyLimitBytes())
}

func TestAppRequestBodyLimit_Set(t *testing.T) {
	app := NewApp()
	require.NoError(t, app.SetRequestBodyLimitBytes(2048))
	require.Equal(t, int64(2048), app.RequestBodyLimitBytes())
}

func TestAppRequestBodyLimit_SetInvalid(t *testing.T) {
	app := NewApp()
	require.Error(t, app.SetRequestBodyLimitBytes(0))
}

func TestAppAdmissionDefaults(t *testing.T) {
	app := NewApp()
	require.Equal(t, DefaultMaxConcurrentRequests, app.MaxConcurrentRequests())
	require.Equal(t, DefaultMaxQueuedRequests, app.MaxQueuedRequests())
	require.Equal(t, DefaultQueueTimeout, app.QueueTimeout())
	require.Equal(t, DefaultReadHeaderTimeout, app.ReadHeaderTimeout())
	require.Equal(t, DefaultReadTimeout, app.ReadTimeout())
	require.Equal(t, DefaultWriteTimeout, app.WriteTimeout())
	require.Equal(t, DefaultIdleTimeout, app.IdleTimeout())
}

func TestAppAdmissionSetters(t *testing.T) {
	app := NewApp()
	require.NoError(t, app.SetMaxConcurrentRequests(3))
	require.NoError(t, app.SetMaxQueuedRequests(4))
	require.NoError(t, app.SetQueueTimeout(150*time.Millisecond))
	require.NoError(t, app.SetReadHeaderTimeout(2*time.Second))
	require.NoError(t, app.SetReadTimeout(10*time.Second))
	require.NoError(t, app.SetWriteTimeout(15*time.Second))
	require.NoError(t, app.SetIdleTimeout(45*time.Second))

	require.Equal(t, 3, app.MaxConcurrentRequests())
	require.Equal(t, 4, app.MaxQueuedRequests())
	require.Equal(t, 150*time.Millisecond, app.QueueTimeout())
	require.Equal(t, 2*time.Second, app.ReadHeaderTimeout())
	require.Equal(t, 10*time.Second, app.ReadTimeout())
	require.Equal(t, 15*time.Second, app.WriteTimeout())
	require.Equal(t, 45*time.Second, app.IdleTimeout())
}

func TestAppAdmissionSettersRejectInvalid(t *testing.T) {
	app := NewApp()
	require.Error(t, app.SetMaxConcurrentRequests(0))
	require.Error(t, app.SetMaxQueuedRequests(0))
	require.Error(t, app.SetQueueTimeout(0))
	require.Error(t, app.SetReadHeaderTimeout(0))
	require.Error(t, app.SetReadTimeout(0))
	require.Error(t, app.SetWriteTimeout(0))
	require.Error(t, app.SetIdleTimeout(0))
}
