/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	appI18n "github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEmailLoginTest(t *testing.T) *gorm.DB {
	t.Helper()
	require.NoError(t, appI18n.Init())
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousRedis := common.RedisEnabled
	previousSessionSecret := common.SessionSecret
	previousEmailLoginVerification := common.EmailLoginVerificationEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.AuthFlow{},
		&model.TwoFA{},
		&model.UserSession{},
		&model.Log{},
	))
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.SessionSecret = "email-login-test-session-secret"
	common.EmailLoginVerificationEnabled = true
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.RedisEnabled = previousRedis
		common.SessionSecret = previousSessionSecret
		common.EmailLoginVerificationEnabled = previousEmailLoginVerification
	})
	return db
}

func TestPasswordLoginRequiresEmailVerificationBeforeCreatingSession(t *testing.T) {
	db := setupEmailLoginTest(t)
	passwordHash, err := common.Password2Hash("correct-password")
	require.NoError(t, err)
	user := &model.User{
		Username: "email-login-user", Password: passwordHash, Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)

	router := gin.New()
	router.POST("/api/user/login", Login)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/user/login",
		strings.NewReader(`{"username":"email-login-user","password":"correct-password"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			RequireEmailVerification bool   `json:"require_email_verification"`
			EmailRequired            bool   `json:"email_required"`
			FlowToken                string `json:"flow_token"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	assert.True(t, payload.Data.RequireEmailVerification)
	assert.True(t, payload.Data.EmailRequired)
	assert.NotEmpty(t, payload.Data.FlowToken)
	var sessionCount int64
	require.NoError(t, db.Model(&model.UserSession{}).Count(&sessionCount).Error)
	assert.Zero(t, sessionCount)
}

func TestEmailLoginVerificationBindsFirstEmailAndRejectsReplay(t *testing.T) {
	db := setupEmailLoginTest(t)
	user := &model.User{
		Username: "email-enrollment-user", Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)

	const email = "first-login@example.com"
	const code = "a1b2c3"
	payloadBytes, err := common.Marshal(emailLoginFlowPayload{
		AuthVersion: user.AuthVersion,
		Email:       email,
		CodeHash:    emailLoginVerificationCodeHash(email, code),
	})
	require.NoError(t, err)
	flowToken, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose: model.AuthFlowPurposeEmailLogin, UserId: user.Id,
		Payload: string(payloadBytes), ExpiresAt: time.Now().Add(time.Minute),
	})
	require.NoError(t, err)

	router := gin.New()
	router.POST("/api/user/login/email/verify", VerifyEmailLogin)
	verify := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/user/login/email/verify",
			strings.NewReader(`{"flow_token":"`+flowToken+`","code":"`+code+`"}`),
		)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}

	firstResponse := verify()
	var firstPayload struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(firstResponse.Body.Bytes(), &firstPayload))
	require.True(t, firstPayload.Success)
	assert.NotEmpty(t, firstPayload.Data.AccessToken)
	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, email, stored.Email)

	replayResponse := verify()
	var replayPayload struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(replayResponse.Body.Bytes(), &replayPayload))
	assert.False(t, replayPayload.Success)
}
