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
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from '@/components/ui/input-otp'
import { Spinner } from '@/components/ui/spinner'

interface EmailLoginVerificationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  emailRequired: boolean
  maskedEmail: string
  email: string
  code: string
  onEmailChange: (value: string) => void
  onCodeChange: (value: string) => void
  onSend: () => void
  onVerify: () => void
  isSending: boolean
  isVerifying: boolean
  secondsLeft: number
}

export function EmailLoginVerificationDialog(
  props: EmailLoginVerificationDialogProps
) {
  const { t } = useTranslation()
  const sendDisabled =
    props.isSending ||
    props.isVerifying ||
    props.secondsLeft > 0 ||
    (props.emailRequired && !props.email.trim())
  const sendLabel =
    props.secondsLeft > 0
      ? t('Resend ({{seconds}}s)', { seconds: props.secondsLeft })
      : t('Send code')
  const busy = props.isSending || props.isVerifying
  const emailLocked = props.emailRequired && props.maskedEmail !== ''

  function handleOpenChange(open: boolean) {
    if (!open && busy) return
    props.onOpenChange(open)
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={handleOpenChange}
      title={t('Login Email Verification')}
      description={
        props.emailRequired
          ? t('This account has no email address. Verify one now to continue.')
          : t('Enter the code sent to your email to finish signing in.')
      }
      contentClassName='max-w-sm'
      contentHeight='auto'
      bodyClassName='space-y-4'
      showCloseButton={!busy}
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSending || props.isVerifying}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onVerify}
            disabled={
              props.isSending ||
              props.isVerifying ||
              props.code.trim().length !== 6
            }
            className='gap-2'
          >
            {props.isVerifying ? <Spinner data-icon='inline-start' /> : null}
            {t('Verify and sign in')}
          </Button>
        </>
      }
    >
      <FieldGroup className='gap-4'>
        {props.emailRequired ? (
          <Field>
            <FieldLabel htmlFor='login-verification-email'>
              {t('Email address')}
            </FieldLabel>
            <Input
              id='login-verification-email'
              type='email'
              autoComplete='email'
              placeholder={t('name@example.com')}
              value={props.email}
              onChange={(event) => props.onEmailChange(event.target.value)}
              disabled={busy || emailLocked}
            />
          </Field>
        ) : null}

        {props.maskedEmail ? (
          <p className='bg-muted/50 text-muted-foreground rounded-md border px-3 py-2 text-sm'>
            {t('Code sent to {{email}}', { email: props.maskedEmail })}
          </p>
        ) : null}

        <Field>
          <div className='flex items-center justify-between gap-3'>
            <FieldLabel htmlFor='login-verification-code'>
              {t('Verification code')}
            </FieldLabel>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={props.onSend}
              disabled={sendDisabled}
              className='h-7 px-2 text-xs'
            >
              {props.isSending ? (
                <Spinner data-icon='inline-start' />
              ) : (
                sendLabel
              )}
            </Button>
          </div>
          <InputOTP
            id='login-verification-code'
            maxLength={6}
            value={props.code}
            onChange={props.onCodeChange}
            autoComplete='one-time-code'
            containerClassName='justify-center'
            disabled={busy}
          >
            <InputOTPGroup>
              <InputOTPSlot index={0} />
              <InputOTPSlot index={1} />
              <InputOTPSlot index={2} />
              <InputOTPSlot index={3} />
              <InputOTPSlot index={4} />
              <InputOTPSlot index={5} />
            </InputOTPGroup>
          </InputOTP>
        </Field>
      </FieldGroup>
    </Dialog>
  )
}
