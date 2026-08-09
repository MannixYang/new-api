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
const { RechargeFormCard } = await import('../recharge-form-card')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'zh',
  resources: {
    zh: {
      translation: {
        'Add Funds': '添加资金',
        'Offline top-ups and after-sales support are available.':
          '提供线下充值及售后支持。',
        'Online payment is temporarily unavailable during the trial operation.':
          '试运行期间暂不开放在线支付。',
        'Prices for larger amounts are negotiable.': '高额度价格可议。',
        'Private relay deployment is also available with remote setup.':
          '也承接线下中转站搭建，支持远程部署。',
        'Trial operation': '试运行',
        'Trial operation and offline top-up information':
          '试运行与线下充值信息',
        'Trial price': '试运行价格',
        'WeChat contact': '微信联系',
        '¥10 for US$30 in credit': '10 元可充值 30 美元额度',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('wallet add funds card', () => {
  after(() => {
    domWindow.close()
  })

  test('shows trial contact details and hides configured online payment controls', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <RechargeFormCard
            topupInfo={{
              enable_online_topup: true,
              enable_stripe_topup: true,
              pay_methods: [{ name: 'Alipay', type: 'alipay' }],
              min_topup: 1,
              stripe_min_topup: 1,
              amount_options: [10, 30],
              discount: {},
              enable_redemption: true,
            }}
            presetAmounts={[{ value: 10 }]}
            selectedPreset={null}
            onSelectPreset={() => {}}
            topupAmount={10}
            onTopupAmountChange={() => {}}
            paymentAmount={10}
            calculating={false}
            onPaymentMethodSelect={() => {}}
            paymentLoading={null}
            redemptionCode=''
            onRedemptionCodeChange={() => {}}
            onRedeem={() => {}}
            redeeming={false}
            loading
          />
        </I18nextProvider>
      )
    })

    const text = container.textContent ?? ''
    assert.equal(text.includes('试运行期间暂不开放在线支付。'), true)
    assert.equal(text.includes('Mannix_yang_'), true)
    assert.equal(text.includes('10 元可充值 30 美元额度'), true)
    assert.equal(text.includes('高额度价格可议。'), true)
    assert.equal(text.includes('也承接线下中转站搭建，支持远程部署。'), true)
    assert.equal(text.includes('Alipay'), false)
    assert.equal(container.querySelector('input'), null)

    await act(async () => root.unmount())
    container.remove()
  })
})
