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

func TestUpdateOptionRejectsEmailVerificationWithoutCompleteSMTP(t *testing.T) {
	previousServer := common.SMTPServer
	previousPort := common.SMTPPort
	previousAccount := common.SMTPAccount
	previousToken := common.SMTPToken
	previousFrom := common.SMTPFrom
	common.SMTPServer = ""
	common.SMTPPort = 465
	common.SMTPAccount = "sender@example.com"
	common.SMTPToken = "smtp-token"
	common.SMTPFrom = "sender@example.com"
	t.Cleanup(func() {
		common.SMTPServer = previousServer
		common.SMTPPort = previousPort
		common.SMTPAccount = previousAccount
		common.SMTPToken = previousToken
		common.SMTPFrom = previousFrom
	})

	for _, optionKey := range []string{
		"EmailVerificationEnabled",
		"EmailLoginVerificationEnabled",
	} {
		t.Run(optionKey, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(
				http.MethodPut,
				"/api/option/",
				strings.NewReader(`{"key":"`+optionKey+`","value":true}`),
			)

			UpdateOption(context)

			assert.Equal(t, http.StatusOK, response.Code)
			var payload struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.False(t, payload.Success)
			assert.Contains(t, payload.Message, "SMTP")
		})
	}
}
