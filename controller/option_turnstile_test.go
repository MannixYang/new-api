package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionRejectsTurnstileWithoutSecretKey(t *testing.T) {
	previousSiteKey := common.TurnstileSiteKey
	previousSecretKey := common.TurnstileSecretKey
	common.TurnstileSiteKey = "site-key"
	common.TurnstileSecretKey = ""
	t.Cleanup(func() {
		common.TurnstileSiteKey = previousSiteKey
		common.TurnstileSecretKey = previousSecretKey
	})

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/option/",
		strings.NewReader(`{"key":"TurnstileCheckEnabled","value":true}`),
	)

	UpdateOption(context)

	assert.Equal(t, http.StatusOK, response.Code)
	var payload struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	assert.False(t, payload.Success)
	assert.Contains(t, payload.Message, "Turnstile")
}
