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
}

func TestAppAdmissionSetters(t *testing.T) {
	app := NewApp()
	require.NoError(t, app.SetMaxConcurrentRequests(3))
	require.NoError(t, app.SetMaxQueuedRequests(4))
	require.NoError(t, app.SetQueueTimeout(150*time.Millisecond))

	require.Equal(t, 3, app.MaxConcurrentRequests())
	require.Equal(t, 4, app.MaxQueuedRequests())
	require.Equal(t, 150*time.Millisecond, app.QueueTimeout())
}

func TestAppAdmissionSettersRejectInvalid(t *testing.T) {
	app := NewApp()
	require.Error(t, app.SetMaxConcurrentRequests(0))
	require.Error(t, app.SetMaxQueuedRequests(0))
	require.Error(t, app.SetQueueTimeout(0))
}
