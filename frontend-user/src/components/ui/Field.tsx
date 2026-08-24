import type { InputHTMLAttributes, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'

interface FieldWrapProps {
  label: string
  error?: string
  children: React.ReactNode
}

export function FieldWrap({ label, error, children }: FieldWrapProps) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs uppercase tracking-[0.16em] text-mute">{label}</span>
      {children}
      {error ? <p className="mt-1 text-xs text-copper">{error}</p> : null}
    </label>
  )
}

const control =
  'w-full rounded-sm border border-foam/15 bg-ink/60 px-3 py-2 text-foam outline-none focus:border-sonar/60'

export function TextInput({
  label,
  error,
  ...rest
}: InputHTMLAttributes<HTMLInputElement> & { label: string; error?: string }) {
  return (
    <FieldWrap label={label} error={error}>
      <input className={control} {...rest} />
    </FieldWrap>
  )
}

export function TextArea({
  label,
  error,
  ...rest
}: TextareaHTMLAttributes<HTMLTextAreaElement> & { label: string; error?: string }) {
  return (
    <FieldWrap label={label} error={error}>
      <textarea className={`${control} min-h-[88px]`} {...rest} />
    </FieldWrap>
  )
}

export function Select({
  label,
  error,
  children,
  ...rest
}: SelectHTMLAttributes<HTMLSelectElement> & { label: string; error?: string }) {
  return (
    <FieldWrap label={label} error={error}>
      <select className={`lm-select ${control}`} {...rest}>
        {children}
      </select>
    </FieldWrap>
  )
}
