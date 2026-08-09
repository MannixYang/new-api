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
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const emailLoginFlowDuration = 10 * time.Minute

var (
	errEmailLoginDomainRestricted = errors.New("email domain is not allowed")
	errEmailLoginAliasRestricted  = errors.New("email alias is not allowed")
)

type emailLoginFlowPayload struct {
	AuthVersion int64  `json:"auth_version"`
	Email       string `json:"email,omitempty"`
	CodeHash    string `json:"code_hash,omitempty"`
}

type emailLoginSendRequest struct {
	FlowToken string `json:"flow_token"`
	Email     string `json:"email,omitempty"`
}

type emailLoginVerifyRequest struct {
	FlowToken string `json:"flow_token"`
	Code      string `json:"code"`
}

func emailLoginVerificationCodeHash(email string, code string) string {
	normalized := model.NormalizeEmail(email) + ":" + strings.ToLower(strings.TrimSpace(code))
	return common.GenerateHMACWithKey([]byte("email-login-code-v1:"+common.SessionSecret), normalized)
}

func maskEmailAddress(email string) string {
	email = model.NormalizeEmail(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	visible := 1
	if len(parts[0]) > 4 {
		visible = 2
	}
	return parts[0][:visible] + "***@" + parts[1]
}

func validateEmailLoginTarget(email string, userId int) error {
	email = model.NormalizeEmail(email)
	if err := common.Validate.Var(email, "required,email"); err != nil {
		return err
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return errors.New("invalid email address")
	}
	if common.EmailDomainRestrictionEnabled {
		allowed := false
		for _, domain := range common.EmailDomainWhitelist {
			if strings.EqualFold(strings.TrimSpace(domain), parts[1]) {
				allowed = true
				break
			}
		}
		if !allowed {
			return errEmailLoginDomainRestricted
		}
	}
	if common.EmailAliasRestrictionEnabled &&
		(strings.Contains(parts[0], "+") || strings.Contains(parts[0], ".")) {
		return errEmailLoginAliasRestricted
	}
	return model.EnsureEmailAvailable(email, userId)
}

func createEmailLoginFlow(user *model.User, email string, codeHash string) (string, int64, error) {
	if user == nil || user.Id <= 0 || user.AuthVersion <= 0 {
		return "", 0, model.ErrAuthFlowInvalid
	}
	payload, err := common.Marshal(emailLoginFlowPayload{
		AuthVersion: user.AuthVersion,
		Email:       model.NormalizeEmail(email),
		CodeHash:    codeHash,
	})
	if err != nil {
		return "", 0, err
	}
	expiresAt := time.Now().Add(emailLoginFlowDuration)
	token, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeEmailLogin,
		UserId:    user.Id,
		Payload:   string(payload),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", 0, err
	}
	return token, expiresAt.Unix(), nil
}

func getEmailLoginFlow(flowToken string) (*model.AuthFlow, *model.User, emailLoginFlowPayload, error) {
	flow, err := model.GetAuthFlow(flowToken, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeEmailLogin})
	if err != nil {
		return nil, nil, emailLoginFlowPayload{}, err
	}
	user, err := model.GetUserById(flow.UserId, false)
	if err != nil {
		return nil, nil, emailLoginFlowPayload{}, err
	}
	var payload emailLoginFlowPayload
	if err := common.UnmarshalJsonStr(flow.Payload, &payload); err != nil ||
		payload.AuthVersion <= 0 ||
		payload.AuthVersion != user.AuthVersion ||
		user.Status != common.UserStatusEnabled {
		return nil, nil, emailLoginFlowPayload{}, model.ErrAuthFlowInvalid
	}
	return flow, user, payload, nil
}

func sendEmailLoginCode(email string, code string) error {
	subject := fmt.Sprintf("%s 登录验证码", common.SystemName)
	content := fmt.Sprintf("<p>您好，您正在登录 %s。</p>"+
		"<p>验证码为：<strong>%s</strong></p>"+
		"<p>验证码 %d 分钟内有效。如非本人操作，请立即修改密码。</p>",
		common.SystemName, code, common.VerificationValidMinutes)
	return common.SendEmail(subject, email, content)
}

func beginEmailLoginVerification(user *model.User, c *gin.Context) {
	token, expiresAt, err := createEmailLoginFlow(user, user.Email, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(c, i18n.MsgUserEmailVerificationRequired),
		"success": true,
		"data": gin.H{
			"require_email_verification": true,
			"flow_token":                 token,
			"email_required":             model.NormalizeEmail(user.Email) == "",
			"masked_email":               maskEmailAddress(user.Email),
			"expires_at":                 expiresAt,
		},
	})
}

func writeEmailLoginFlowExpired(c *gin.Context) {
	common.ApiErrorI18n(c, i18n.MsgUserLoginFlowExpired)
}

func writeEmailLoginTargetError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrEmailAlreadyTaken):
		common.ApiErrorI18n(c, i18n.MsgUserEmailAlreadyTaken)
	case errors.Is(err, errEmailLoginDomainRestricted):
		common.ApiErrorMsg(c, "该邮箱域名不在管理员允许范围内")
	case errors.Is(err, errEmailLoginAliasRestricted):
		common.ApiErrorMsg(c, "管理员已禁止使用邮箱别名")
	default:
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
	}
}

func SendEmailLoginVerification(c *gin.Context) {
	var req emailLoginSendRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	flow, user, payload, err := getEmailLoginFlow(req.FlowToken)
	if err != nil {
		writeEmailLoginFlowExpired(c)
		return
	}
	targetEmail := model.NormalizeEmail(user.Email)
	if targetEmail == "" {
		targetEmail = model.NormalizeEmail(req.Email)
		if targetEmail == "" {
			targetEmail = model.NormalizeEmail(payload.Email)
		}
	}
	if err := validateEmailLoginTarget(targetEmail, user.Id); err != nil {
		writeEmailLoginTargetError(c, err)
		return
	}

	code := common.GenerateVerificationCode(6)
	codeHash := emailLoginVerificationCodeHash(targetEmail, code)
	newToken, expiresAt, err := createEmailLoginFlow(user, targetEmail, codeHash)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := sendEmailLoginCode(targetEmail, code); err != nil {
		_, _ = model.ConsumeAuthFlow(newToken, model.AuthFlowMatch{
			Purpose: model.AuthFlowPurposeEmailLogin,
			UserId:  user.Id,
		})
		common.ApiError(c, err)
		return
	}
	if _, err := model.ConsumeAuthFlow(req.FlowToken, model.AuthFlowMatch{
		Purpose: model.AuthFlowPurposeEmailLogin,
		UserId:  flow.UserId,
	}); err != nil {
		_, _ = model.ConsumeAuthFlow(newToken, model.AuthFlowMatch{
			Purpose: model.AuthFlowPurposeEmailLogin,
			UserId:  user.Id,
		})
		writeEmailLoginFlowExpired(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"flow_token":   newToken,
			"masked_email": maskEmailAddress(targetEmail),
			"expires_at":   expiresAt,
		},
	})
}

func VerifyEmailLogin(c *gin.Context) {
	var req emailLoginVerifyRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	_, user, payload, err := getEmailLoginFlow(req.FlowToken)
	if err != nil || payload.Email == "" || payload.CodeHash == "" {
		writeEmailLoginFlowExpired(c)
		return
	}
	actualHash := emailLoginVerificationCodeHash(payload.Email, req.Code)
	if subtle.ConstantTimeCompare([]byte(actualHash), []byte(payload.CodeHash)) != 1 {
		common.ApiErrorI18n(c, i18n.MsgUserVerificationCodeError)
		return
	}

	boundEmail := model.NormalizeEmail(user.Email)
	if boundEmail != "" && boundEmail != payload.Email {
		writeEmailLoginFlowExpired(c)
		return
	}
	if boundEmail == "" {
		if err := validateEmailLoginTarget(payload.Email, user.Id); err != nil {
			writeEmailLoginTargetError(c, err)
			return
		}
	}

	emailWasBound := boundEmail == ""
	if _, err := model.ConsumeAuthFlowWithAction(req.FlowToken, model.AuthFlowMatch{
		Purpose: model.AuthFlowPurposeEmailLogin,
		UserId:  user.Id,
	}, func(tx *gorm.DB, _ *model.AuthFlow) error {
		if !emailWasBound {
			return nil
		}
		return model.BindEmailToUnboundUserWithTx(tx, user.Id, payload.Email)
	}); err != nil {
		if errors.Is(err, model.ErrEmailAlreadyTaken) {
			common.ApiErrorI18n(c, i18n.MsgUserEmailAlreadyTaken)
			return
		}
		writeEmailLoginFlowExpired(c)
		return
	}
	if emailWasBound {
		if err := model.PublishUserAuthCache(user.Id); err != nil {
			common.SysLog(fmt.Sprintf("failed to publish email binding for user %d: %v", user.Id, err))
		}
	}
	user, err = model.GetUserById(user.Id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	continuePasswordLogin(user, payload.AuthVersion, c)
}

func continuePasswordLogin(user *model.User, expectedAuthVersion int64, c *gin.Context) {
	twoFAEnabled, err := model.IsTwoFAEnabled(user.Id)
	if err != nil {
		common.SysLog(fmt.Sprintf("Login failed to load 2FA status for user %d: %v", user.Id, err))
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	if !twoFAEnabled {
		setupLoginAtAuthVersion(user, expectedAuthVersion, c)
		return
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	payload, err := common.Marshal(twoFALoginFlowPayload{AuthVersion: expectedAuthVersion})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	flowToken, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeTwoFALogin,
		UserId:    user.Id,
		Payload:   string(payload),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(c, i18n.MsgUserRequire2FA),
		"success": true,
		"data": gin.H{
			"require_2fa": true,
			"flow_token":  flowToken,
			"expires_at":  expiresAt.Unix(),
		},
	})
}
