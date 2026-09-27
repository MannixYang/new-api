package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type turnstileRoundTripFunc func(*http.Request) (*http.Response, error)

func (f turnstileRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func withTurnstileTestConfig(t *testing.T, roundTrip turnstileRoundTripFunc) {
	t.Helper()
	previousEnabled := common.TurnstileCheckEnabled
	previousSecret := common.TurnstileSecretKey
	previousClient := turnstileHTTPClient
	common.TurnstileCheckEnabled = true
	common.TurnstileSecretKey = "test-secret"
	turnstileHTTPClient = &http.Client{Transport: roundTrip}
	t.Cleanup(func() {
		common.TurnstileCheckEnabled = previousEnabled
		common.TurnstileSecretKey = previousSecret
		turnstileHTTPClient = previousClient
	})
}

func TestTurnstileCheckRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	withTurnstileTestConfig(t, func(_ *http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})

	router := gin.New()
	router.POST("/login", TurnstileCheck(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "Turnstile token 为空")
	assert.False(t, called)
}

func TestTurnstileCheckUsesHeaderTokenAndFormBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withTurnstileTestConfig(t, func(request *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		form, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		assert.Equal(t, "test-secret", form.Get("secret"))
		assert.Equal(t, "header-token", form.Get("response"))
		assert.Equal(t, "192.0.2.10", form.Get("remoteip"))
		assert.Empty(t, request.URL.RawQuery)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
			Header:     make(http.Header),
		}, nil
	})

	router := gin.New()
	router.POST("/login", TurnstileCheck(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	request.RemoteAddr = "192.0.2.10:12345"
	request.Header.Set("X-Turnstile-Token", "header-token")
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusNoContent, response.Code)
}

func TestTurnstileCheckRejectsUnsuccessfulVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withTurnstileTestConfig(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"success":false}`)),
			Header:     make(http.Header),
		}, nil
	})

	router := gin.New()
	router.POST("/login", TurnstileCheck(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login?turnstile=query-token", nil)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), turnstileFailureMessage)
}
