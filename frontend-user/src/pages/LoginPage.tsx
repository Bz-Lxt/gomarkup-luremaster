import { useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { ApiError } from '../api/client'
import { Button } from '../components/ui/Button'
import { TextInput } from '../components/ui/Field'
import { useAuth } from '../context/AuthContext'
import { useToast } from '../context/ToastContext'

const DEMO_EMAIL = 'hunter@lure.local'
const DEMO_PASS = 'LureHunt@2026'

export function LoginPage() {
  const { user, ready, login } = useAuth()
  const { toast } = useToast()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  if (ready && user) return <Navigate to="/atlas" replace />

  async function submit() {
    const e: Record<string, string> = {}
    if (!email.trim()) e.email = '邮箱必填'
    if (!password) e.password = '密码必填'
    setErrors(e)
    if (Object.keys(e).length) return
    setBusy(true)
    try {
      await login(email.trim(), password)
      toast('欢迎回港', 'ok')
      navigate('/atlas')
    } catch (err) {
      toast(err instanceof ApiError ? err.message : '登录失败', 'err')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="chart-grain flex min-h-screen w-full items-center justify-center px-4 py-10">
      <div className="w-full max-w-md panel p-8">
        <p className="font-mono text-[10px] tracking-[0.28em] text-sonar">LUREMASTER</p>
        <h1 className="mt-2 font-display text-4xl italic leading-tight">夜航海图室</h1>
        <p className="mt-2 text-sm text-mute">把作战室摊开在灯下。铜钉、潮汐、一口鱼。</p>

        <div className="mt-8 space-y-4">
          <TextInput label="邮箱" type="email" value={email} onChange={(e) => setEmail(e.target.value)} error={errors.email} />
          <TextInput
            label="密码"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={errors.password}
          />
          <Button variant="sonar" className="w-full" disabled={busy} onClick={submit}>
            {busy ? '登舰中…' : '进入海图'}
          </Button>
          <Button
            variant="ghost"
            className="w-full"
            onClick={() => {
              setEmail(DEMO_EMAIL)
              setPassword(DEMO_PASS)
              toast('已填入测试账号', 'info')
            }}
          >
            一键填入测试账号
          </Button>
        </div>

        <div className="mt-6 rounded-sm border border-foam/10 bg-ink/40 px-3 py-3 font-mono text-xs text-mute">
          <p>测试账号 {DEMO_EMAIL}</p>
          <p>密码 {DEMO_PASS}</p>
        </div>

        <div className="mt-6 flex justify-between text-xs text-mute">
          <Link to="/privacy" className="hover:text-tide">
            隐私
          </Link>
          <Link to="/terms" className="hover:text-tide">
            条款
          </Link>
        </div>
      </div>
    </div>
  )
}
