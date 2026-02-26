package rstf

import (
	"net/http"
	"time"
)

const (
	DefaultMaxConcurrentRequests = 128
	DefaultMaxQueuedRequests     = 256
	DefaultQueueTimeout          = 5 * time.Second
)

// AdmissionControlConfig configures request admission and backpressure behavior.
type AdmissionControlConfig struct {
	MaxConcurrentRequests int
	MaxQueuedRequests     int
	QueueTimeout          time.Duration
}

type admissionController struct {
	inFlight chan struct{}
	queued   chan struct{}
	timeout  time.Duration
	config   AdmissionControlConfig
}

// NewAdmissionMiddleware returns middleware enforcing deterministic run/enqueue/drop admission.
func NewAdmissionMiddleware(cfg AdmissionControlConfig) Middleware {
	controller := newAdmissionController(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if controller.tryRun() {
				defer controller.finish()
				next.ServeHTTP(w, req)
				return
			}

			if !controller.tryEnqueue() {
				writeOverload(w, controller.config, "queue_full")
				return
			}
			defer controller.dequeue()

			timer := time.NewTimer(controller.timeout)
			defer timer.Stop()

			select {
			case controller.inFlight <- struct{}{}:
				defer controller.finish()
				next.ServeHTTP(w, req)
				return
			case <-timer.C:
				writeOverload(w, controller.config, "queue_timeout")
				return
			case <-req.Context().Done():
				return
			}
		})
	}
}

func newAdmissionController(cfg AdmissionControlConfig) *admissionController {
	normalized := AdmissionControlConfig{
		MaxConcurrentRequests: cfg.MaxConcurrentRequests,
		MaxQueuedRequests:     cfg.MaxQueuedRequests,
		QueueTimeout:          cfg.QueueTimeout,
	}
	if normalized.MaxConcurrentRequests <= 0 {
		normalized.MaxConcurrentRequests = DefaultMaxConcurrentRequests
	}
	if normalized.MaxQueuedRequests <= 0 {
		normalized.MaxQueuedRequests = DefaultMaxQueuedRequests
	}
	if normalized.QueueTimeout <= 0 {
		normalized.QueueTimeout = DefaultQueueTimeout
	}

	return &admissionController{
		inFlight: make(chan struct{}, normalized.MaxConcurrentRequests),
		queued:   make(chan struct{}, normalized.MaxQueuedRequests),
		timeout:  normalized.QueueTimeout,
		config:   normalized,
	}
}

func (a *admissionController) tryRun() bool {
	select {
	case a.inFlight <- struct{}{}:
		return true
	default:
		return false
	}
}

func (a *admissionController) finish() {
	<-a.inFlight
}

func (a *admissionController) tryEnqueue() bool {
	select {
	case a.queued <- struct{}{}:
		return true
	default:
		return false
	}
}

func (a *admissionController) dequeue() {
	<-a.queued
}

func writeOverload(w http.ResponseWriter, cfg AdmissionControlConfig, reason string) {
	WriteErrorEnvelope(w, &RequestError{
		Code:    ErrorCodeOverloaded,
		Message: "server overloaded",
		Details: map[string]any{
			"reason":                reason,
			"maxConcurrentRequests": cfg.MaxConcurrentRequests,
			"maxQueuedRequests":     cfg.MaxQueuedRequests,
			"queueTimeoutMs":        cfg.QueueTimeout.Milliseconds(),
		},
		Status: http.StatusServiceUnavailable,
	})
}
