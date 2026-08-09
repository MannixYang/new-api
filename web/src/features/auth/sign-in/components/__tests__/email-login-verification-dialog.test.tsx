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
  'HTMLButtonElement',
  'HTMLInputElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'KeyboardEvent',
  'PointerEvent',
  'MouseEvent',
  'FocusEvent',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
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
const { EmailLoginVerificationDialog } =
  await import('../email-login-verification-dialog')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'zh',
  resources: {
    zh: {
      translation: {
        Cancel: '取消',
        'Code sent to {{email}}': '验证码已发送至 {{email}}',
        'Email address': '邮箱地址',
        'Enter the code sent to your email to finish signing in.':
          '输入邮件中的验证码以完成登录。',
        'Login Email Verification': '登录邮箱验证',
        'Resend ({{seconds}}s)': '{{seconds}} 秒后重发',
        'Send code': '发送验证码',
        'This account has no email address. Verify one now to continue.':
          '该账号尚未绑定邮箱，请验证邮箱后继续。',
        'Verification code': '验证码',
        'Verify and sign in': '验证并登录',
        'name@example.com': 'name@example.com',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('email login verification dialog', () => {
  after(() => {
    domWindow.close()
  })

  test('requires an email address when an existing account has none', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <EmailLoginVerificationDialog
            open
            onOpenChange={() => {}}
            emailRequired
            maskedEmail=''
            email=''
            code=''
            onEmailChange={() => {}}
            onCodeChange={() => {}}
            onSend={() => {}}
            onVerify={() => {}}
            isSending={false}
            isVerifying={false}
            secondsLeft={0}
          />
        </I18nextProvider>
      )
    })

    const emailInput = document.querySelector<HTMLInputElement>(
      '#login-verification-email'
    )
    const text = document.body.textContent ?? ''
    assert.ok(emailInput)
    assert.equal(text.includes('该账号尚未绑定邮箱'), true)
    const sendButton = [...document.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('发送验证码')
    )
    assert.ok(sendButton)
    assert.equal(sendButton.disabled, true)

    await act(async () => root.unmount())
    container.remove()
  })

  test('shows the masked destination for an account with a bound email', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <EmailLoginVerificationDialog
            open
            onOpenChange={() => {}}
            emailRequired={false}
            maskedEmail='us***@example.com'
            email=''
            code='a1b2c3'
            onEmailChange={() => {}}
            onCodeChange={() => {}}
            onSend={() => {}}
            onVerify={() => {}}
            isSending={false}
            isVerifying={false}
            secondsLeft={20}
          />
        </I18nextProvider>
      )
    })

    const text = document.body.textContent ?? ''
    assert.equal(text.includes('us***@example.com'), true)
    assert.equal(document.querySelector('#login-verification-email'), null)
    const verifyButton = [...document.querySelectorAll('button')].find(
      (button) => button.textContent?.includes('验证并登录')
    )
    assert.ok(verifyButton)
    assert.equal(verifyButton.disabled, false)

    await act(async () => root.unmount())
    container.remove()
  })

  test('prevents dismissal and disables inputs while sending a code', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <EmailLoginVerificationDialog
            open
            onOpenChange={() => {}}
            emailRequired
            maskedEmail=''
            email='user@example.com'
            code=''
            onEmailChange={() => {}}
            onCodeChange={() => {}}
            onSend={() => {}}
            onVerify={() => {}}
            isSending
            isVerifying={false}
            secondsLeft={0}
          />
        </I18nextProvider>
      )
    })

    const emailInput = document.querySelector<HTMLInputElement>(
      '#login-verification-email'
    )
    const codeInput = document.querySelector<HTMLInputElement>(
      '#login-verification-code'
    )
    const cancelButton = [...document.querySelectorAll('button')].find(
      (button) => button.textContent?.includes('取消')
    )

    assert.ok(emailInput)
    assert.ok(codeInput)
    assert.ok(cancelButton)
    assert.equal(emailInput.disabled, true)
    assert.equal(codeInput.disabled, true)
    assert.equal(cancelButton.disabled, true)
    assert.equal(document.body.textContent?.includes('Close'), false)

    await act(async () => root.unmount())
    container.remove()
  })

  test('locks a newly entered email after the code is sent', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <EmailLoginVerificationDialog
            open
            onOpenChange={() => {}}
            emailRequired
            maskedEmail='us***@example.com'
            email='user@example.com'
            code=''
            onEmailChange={() => {}}
            onCodeChange={() => {}}
            onSend={() => {}}
            onVerify={() => {}}
            isSending={false}
            isVerifying={false}
            secondsLeft={20}
          />
        </I18nextProvider>
      )
    })

    const emailInput = document.querySelector<HTMLInputElement>(
      '#login-verification-email'
    )
    assert.ok(emailInput)
    assert.equal(emailInput.disabled, true)
    assert.equal(document.body.textContent?.includes('us***@example.com'), true)

    await act(async () => root.unmount())
    container.remove()
  })
})
