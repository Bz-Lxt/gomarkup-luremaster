import type { ButtonHTMLAttributes } from 'react'

type Variant = 'sonar' | 'copper' | 'ghost' | 'tide'

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
}

const styles: Record<Variant, string> = {
  sonar: 'bg-sonar text-ink hover:brightness-110',
  copper: 'bg-copper text-ink hover:brightness-110',
  tide: 'bg-tide/15 text-tide border border-tide/40 hover:bg-tide/25',
  ghost: 'bg-transparent text-foam border border-foam/15 hover:border-sonar/50 hover:text-sonar',
}

export function Button({ variant = 'sonar', className = '', disabled, ...rest }: Props) {
  return (
    <button
      type="button"
      disabled={disabled}
      className={`inline-flex items-center justify-center gap-2 rounded-sm px-4 py-2 font-medium tracking-wide transition disabled:cursor-not-allowed disabled:opacity-40 ${styles[variant]} ${className}`}
      {...rest}
    />
  )
}
