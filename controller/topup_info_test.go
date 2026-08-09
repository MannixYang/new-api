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
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTopUpInfoReturnsEffectiveReferralRewards(t *testing.T) {
	paymentSetting := operation_setting.GetPaymentSetting()
	originalComplianceConfirmed := paymentSetting.ComplianceConfirmed
	originalComplianceTermsVersion := paymentSetting.ComplianceTermsVersion
	originalInviterReward := common.QuotaForInviter
	originalInviteeReward := common.QuotaForInvitee
	t.Cleanup(func() {
		paymentSetting.ComplianceConfirmed = originalComplianceConfirmed
		paymentSetting.ComplianceTermsVersion = originalComplianceTermsVersion
		common.QuotaForInviter = originalInviterReward
		common.QuotaForInvitee = originalInviteeReward
	})

	common.QuotaForInviter = 500_000
	common.QuotaForInvitee = 250_000

	tests := []struct {
		name                string
		complianceConfirmed bool
		wantInviterReward   int
		wantInviteeReward   int
	}{
		{
			name:                "confirmed settings expose configured rewards",
			complianceConfirmed: true,
			wantInviterReward:   500_000,
			wantInviteeReward:   250_000,
		},
		{
			name:                "unconfirmed settings report disabled rewards",
			complianceConfirmed: false,
			wantInviterReward:   0,
			wantInviteeReward:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paymentSetting.ComplianceConfirmed = tt.complianceConfirmed
			paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)

			GetTopUpInfo(context)

			var payload struct {
				Success bool `json:"success"`
				Data    struct {
					InviterRewardQuota int `json:"inviter_reward_quota"`
					InviteeRewardQuota int `json:"invitee_reward_quota"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			require.True(t, payload.Success)
			assert.Equal(t, tt.wantInviterReward, payload.Data.InviterRewardQuota)
			assert.Equal(t, tt.wantInviteeReward, payload.Data.InviteeRewardQuota)
		})
	}
}
