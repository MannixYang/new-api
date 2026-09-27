package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	turnstileVerifyURL      = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	turnstileFailureMessage = "Turnstile 校验失败，请刷新重试！"
)

type turnstileCheckResponse struct {
	Success bool `json:"success"`
}

var turnstileHTTPClient = &http.Client{Timeout: 10 * time.Second}

func turnstileToken(c *gin.Context) string {
	if token := c.GetHeader("X-Turnstile-Token"); token != "" {
		return token
	}
	return c.Query("turnstile")
}

func verifyTurnstile(ctx context.Context, token string, remoteIP string) error {
	if strings.TrimSpace(common.TurnstileSecretKey) == "" {
		return errors.New("turnstile secret key is not configured")
	}

	form := url.Values{
		"secret":   {common.TurnstileSecretKey},
		"response": {token},
		"remoteip": {remoteIP},
	}
	// Keep the token in the form body instead of the URL so it is not exposed in
	// intermediary request logs.
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := turnstileHTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return errors.New("turnstile verification returned a non-success status")
	}

	var result turnstileCheckResponse
	if err := common.DecodeJson(response.Body, &result); err != nil {
		return err
	}
	if !result.Success {
		return errors.New("turnstile verification failed")
	}
	return nil
}

func TurnstileCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.TurnstileCheckEnabled {
			token := turnstileToken(c)
			if token == "" {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "Turnstile token 为空",
				})
				c.Abort()
				return
			}
			if err := verifyTurnstile(c.Request.Context(), token, c.ClientIP()); err != nil {
				common.SysLog(err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": turnstileFailureMessage,
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
