package actions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPResponse_SingleStatusCode(t *testing.T) {
	action := &HTTPResponse{}
	config := map[string]interface{}{
		"statusCode": 203,
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rr := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), ResponseWriterKey, rr)

	err := action.Execute(ctx, config)
	assert.NoError(t, err)
	assert.Equal(t, 203, rr.Code)
}

func TestHTTPResponse_StatusCodesWithProbabilities(t *testing.T) {
	action := &HTTPResponse{}
	config := map[string]interface{}{
		"statusCodesWithProbabilities": map[interface{}]interface{}{
			200: 0.7,
			500: 0.2,
			404: 0.1,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rr := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), ResponseWriterKey, rr)

	// Simulate multiple executions to cover different probabilities
	for i := 0; i < 100; i++ {
		rr := httptest.NewRecorder()
		ctx := context.WithValue(context.Background(), ResponseWriterKey, rr)
		err := action.Execute(ctx, config)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 500, 404}, rr.Code)
	}
}

func TestHTTPResponse_InvalidConfig(t *testing.T) {
	action := &HTTPResponse{}
	config := map[string]interface{}{
		"invalidField": 123,
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rr := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), ResponseWriterKey, rr)

	err := action.Execute(ctx, config)
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, rr.Code) // Default to 500 on error
}

func TestHTTPResponse_NoConfig(t *testing.T) {
	action := &HTTPResponse{}
	config := map[string]interface{}{}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rr := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), ResponseWriterKey, rr)

	err := action.Execute(ctx, config)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rr.Code) // Default to 200 if no status code is specified
}

func TestHTTPResponse_InvalidProbabilities(t *testing.T) {
	action := &HTTPResponse{}
	config := map[string]interface{}{
		"statusCodesWithProbabilities": map[interface{}]interface{}{
			200: "invalid",
			500: 0.2,
			404: 0.1,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rr := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), ResponseWriterKey, rr)

	err := action.Execute(ctx, config)
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, rr.Code) // Default to 500 on error
}
