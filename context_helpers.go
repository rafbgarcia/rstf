package rstf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const DefaultBodyLimit int64 = 1 << 20

type ErrorCode string

const (
	ErrorCodeInvalidPayload         ErrorCode = "invalid_payload"
	ErrorCodePayloadTooLarge        ErrorCode = "payload_too_large"
	ErrorCodeUnsupportedContentType ErrorCode = "unsupported_content_type"
	ErrorCodeValidationFailed       ErrorCode = "validation_failed"
	ErrorCodeOverloaded             ErrorCode = "overloaded"
	ErrorCodeInternal               ErrorCode = "internal_error"
)

type RequestError struct {
	Code    ErrorCode
	Message string
	Details map[string]any
	Status  int
}

func (e *RequestError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func ValidationError(message string, details map[string]any) error {
	return &RequestError{
		Code:    ErrorCodeValidationFailed,
		Message: message,
		Details: details,
		Status:  http.StatusUnprocessableEntity,
	}
}

func (c *Context) BindJSON(target any) error {
	if c == nil || c.Request == nil {
		return &RequestError{
			Code:    ErrorCodeInternal,
			Message: "request context is not initialized",
			Status:  http.StatusInternalServerError,
		}
	}

	mediaType, err := parseContentType(c.Request.Header.Get("Content-Type"))
	if err != nil {
		return &RequestError{
			Code:    ErrorCodeUnsupportedContentType,
			Message: err.Error(),
			Status:  http.StatusUnsupportedMediaType,
		}
	}
	if !isJSONMediaType(mediaType) {
		return &RequestError{
			Code:    ErrorCodeUnsupportedContentType,
			Message: "Content-Type must be application/json",
			Status:  http.StatusUnsupportedMediaType,
		}
	}

	reader := io.Reader(c.Request.Body)
	limit := c.RequestBodyLimitBytes()
	if c.Writer != nil {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		reader = c.Request.Body
	}

	dec := json.NewDecoder(reader)
	if err := dec.Decode(target); err != nil {
		return mapDecodeError(err, limit)
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return &RequestError{
			Code:    ErrorCodeInvalidPayload,
			Message: "request body must contain exactly one JSON value",
			Status:  http.StatusBadRequest,
		}
	}

	return nil
}

func (c *Context) JSON(status int, payload any) error {
	if c == nil || c.Writer == nil {
		return &RequestError{
			Code:    ErrorCodeInternal,
			Message: "response writer is not initialized",
			Status:  http.StatusInternalServerError,
		}
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(status)
	return json.NewEncoder(c.Writer).Encode(payload)
}

func (c *Context) Text(status int, body string) error {
	if c == nil || c.Writer == nil {
		return &RequestError{
			Code:    ErrorCodeInternal,
			Message: "response writer is not initialized",
			Status:  http.StatusInternalServerError,
		}
	}

	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(status)
	_, err := io.WriteString(c.Writer, body)
	return err
}

func (c *Context) Redirect(status int, location string) error {
	if c == nil || c.Writer == nil || c.Request == nil {
		return &RequestError{
			Code:    ErrorCodeInternal,
			Message: "request context is not initialized",
			Status:  http.StatusInternalServerError,
		}
	}

	http.Redirect(c.Writer, c.Request, location, status)
	return nil
}

func (c *Context) NoContent() error {
	if c == nil || c.Writer == nil {
		return &RequestError{
			Code:    ErrorCodeInternal,
			Message: "response writer is not initialized",
			Status:  http.StatusInternalServerError,
		}
	}

	c.Writer.WriteHeader(http.StatusNoContent)
	return nil
}

func WriteErrorEnvelope(w http.ResponseWriter, err error) {
	status, envelope := ErrorEnvelope(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope)
}

func ErrorEnvelope(err error) (int, map[string]any) {
	re := requestErrorFrom(err)
	return re.Status, map[string]any{
		"error": map[string]any{
			"code":    re.Code,
			"message": re.Message,
			"details": re.Details,
		},
	}
}

func requestErrorFrom(err error) *RequestError {
	if err == nil {
		return &RequestError{
			Code:    ErrorCodeInternal,
			Message: "internal server error",
			Details: map[string]any{},
			Status:  http.StatusInternalServerError,
		}
	}

	var re *RequestError
	if errors.As(err, &re) {
		if re.Details == nil {
			re.Details = map[string]any{}
		}
		if re.Status == 0 {
			re.Status = mapErrorCodeToStatus(re.Code)
		}
		if re.Message == "" {
			re.Message = http.StatusText(re.Status)
		}
		return re
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return &RequestError{
			Code:    ErrorCodePayloadTooLarge,
			Message: "payload exceeds configured limit",
			Details: map[string]any{"limitBytes": maxBytesErr.Limit},
			Status:  http.StatusRequestEntityTooLarge,
		}
	}

	return &RequestError{
		Code:    ErrorCodeInternal,
		Message: "internal server error",
		Details: map[string]any{},
		Status:  http.StatusInternalServerError,
	}
}

func mapErrorCodeToStatus(code ErrorCode) int {
	switch code {
	case ErrorCodeInvalidPayload:
		return http.StatusBadRequest
	case ErrorCodePayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case ErrorCodeUnsupportedContentType:
		return http.StatusUnsupportedMediaType
	case ErrorCodeValidationFailed:
		return http.StatusUnprocessableEntity
	case ErrorCodeOverloaded:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func mapDecodeError(err error, limit int64) error {
	if errors.Is(err, io.EOF) {
		return &RequestError{
			Code:    ErrorCodeInvalidPayload,
			Message: "request body must not be empty",
			Status:  http.StatusBadRequest,
		}
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return &RequestError{
			Code:    ErrorCodePayloadTooLarge,
			Message: "payload exceeds configured limit",
			Details: map[string]any{"limitBytes": limit},
			Status:  http.StatusRequestEntityTooLarge,
		}
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return &RequestError{
			Code:    ErrorCodeInvalidPayload,
			Message: "invalid JSON payload",
			Details: map[string]any{"offset": syntaxErr.Offset},
			Status:  http.StatusBadRequest,
		}
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return &RequestError{
			Code:    ErrorCodeInvalidPayload,
			Message: "invalid JSON payload",
			Details: map[string]any{
				"field":  typeErr.Field,
				"offset": typeErr.Offset,
			},
			Status: http.StatusBadRequest,
		}
	}

	return &RequestError{
		Code:    ErrorCodeInvalidPayload,
		Message: "invalid JSON payload",
		Status:  http.StatusBadRequest,
	}
}

func parseContentType(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("Content-Type must be application/json")
	}
	mediaType, params, err := mime.ParseMediaType(raw)
	if err != nil {
		return "", fmt.Errorf("invalid Content-Type header")
	}

	if q, ok := params["q"]; ok {
		if _, err := strconv.ParseFloat(q, 64); err != nil {
			return "", fmt.Errorf("invalid Content-Type header")
		}
	}
	return strings.ToLower(mediaType), nil
}

func isJSONMediaType(mediaType string) bool {
	if mediaType == "application/json" {
		return true
	}
	return strings.HasSuffix(mediaType, "+json")
}

type ResponseTracker struct {
	writer      http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func NewResponseTracker(w http.ResponseWriter) *ResponseTracker {
	return &ResponseTracker{writer: w}
}

func (w *ResponseTracker) Header() http.Header {
	return w.writer.Header()
}

func (w *ResponseTracker) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = statusCode
	w.wroteHeader = true
	w.writer.WriteHeader(statusCode)
}

func (w *ResponseTracker) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.writer.Write(p)
}

func (w *ResponseTracker) StatusCode() int {
	if !w.wroteHeader {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *ResponseTracker) Written() bool {
	return w.wroteHeader
}

type headWriter struct {
	tracker *ResponseTracker
}

func NewHeadWriter(w *ResponseTracker) http.ResponseWriter {
	return &headWriter{tracker: w}
}

func (w *headWriter) Header() http.Header {
	return w.tracker.Header()
}

func (w *headWriter) WriteHeader(statusCode int) {
	w.tracker.WriteHeader(statusCode)
}

func (w *headWriter) Write(p []byte) (int, error) {
	if !w.tracker.Written() {
		w.tracker.WriteHeader(http.StatusOK)
	}
	return len(p), nil
}
