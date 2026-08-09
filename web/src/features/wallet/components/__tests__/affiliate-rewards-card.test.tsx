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
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { AffiliateRewardsCard } = await import('../affiliate-rewards-card')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'zh',
  resources: {
    zh: {
      translation: {
        'Copy referral link': '复制推荐链接',
        'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.':
          '用户通过您的推荐链接注册后，您即可获得奖励。可随时将累计奖励转入余额。',
        'Invitee Reward': '受邀者奖励',
        'Inviter Reward': '邀请者奖励',
        Invites: '邀请',
        Pending: '待转入',
        'Referral Program': '推荐计划',
        'Total Earned': '累计奖励',
        'Transfer to Balance': '转入余额',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('affiliate rewards card', () => {
  after(() => {
    domWindow.close()
  })

  test('shows the configured inviter and invitee reward amounts with small-value precision', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <AffiliateRewardsCard
            user={{
              id: 1,
              username: 'inviter',
              quota: 0,
              used_quota: 0,
              request_count: 0,
              aff_quota: 0,
              aff_history_quota: 0,
              aff_count: 0,
              group: 'default',
            }}
            affiliateLink='https://example.com/register?aff=test'
            inviterRewardQuota={10}
            inviteeRewardQuota={500_000}
            onTransfer={() => {}}
          />
        </I18nextProvider>
      )
    })

    const text = container.textContent ?? ''
    assert.equal(text.includes('邀请者奖励'), true)
    assert.equal(text.includes('$0.00002'), true)
    assert.equal(text.includes('受邀者奖励'), true)
    assert.equal(text.includes('$1'), true)

    await act(async () => root.unmount())
    container.remove()
  })
})
