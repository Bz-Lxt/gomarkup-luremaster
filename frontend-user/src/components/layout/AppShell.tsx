import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { useToast } from '../../context/ToastContext'

const NAV = [
  { to: '/atlas', label: '海图', mark: 'A' },
  { to: '/catches', label: '战报', mark: 'R' },
  { to: '/catches/new', label: '记口', mark: '+' },
  { to: '/crew', label: '抢位', mark: 'C' },
  { to: '/lures', label: '拟饵', mark: 'L' },
  { to: '/stats', label: '战绩', mark: 'S' },
]

function NavItems({ onNavigate }: { onNavigate?: () => void }) {
  const { logout, user } = useAuth()
  const navigate = useNavigate()
  const { toast } = useToast()

  return (
    <>
      {NAV.map((n) => (
        <NavLink
          key={n.to}
          to={n.to}
          onClick={onNavigate}
          className={({ isActive }) =>
            `flex flex-col items-center gap-1 rounded-sm px-1 py-2 text-[11px] tracking-wide ${
              isActive ? 'bg-sonar/15 text-sonar' : 'text-mute hover:text-foam'
            }`
          }
        >
          <span className="font-display text-lg italic leading-none">{n.mark}</span>
          <span>{n.label}</span>
        </NavLink>
      ))}
      <button
        type="button"
        className="mt-auto flex flex-col items-center gap-1 px-1 py-2 text-[11px] text-mute hover:text-copper"
        onClick={() => {
          logout()
          toast('已离港', 'info')
          onNavigate?.()
          navigate('/')
        }}
      >
        <span className="font-display text-lg italic leading-none">×</span>
        <span>登出</span>
        {user ? <span className="max-w-[64px] truncate text-[10px] text-mute/70">{user.nickname}</span> : null}
      </button>
    </>
  )
}

export function AppShell() {
  const [open, setOpen] = useState(false)

  return (
    <div className="chart-grain w-full min-h-screen">
      <aside className="fixed left-0 top-0 z-40 hidden h-screen w-[72px] flex-col items-center gap-2 border-r border-foam/10 bg-kelp/90 py-4 backdrop-blur md:flex">
        <div className="mb-2 font-display text-2xl italic text-sonar">LM</div>
        <NavItems />
      </aside>

      <header className="sticky top-0 z-40 flex items-center justify-between border-b border-foam/10 bg-kelp/90 px-4 py-3 backdrop-blur md:hidden">
        <span className="font-display text-xl italic text-sonar">夜航海图室</span>
        <button
          type="button"
          className="rounded-sm border border-foam/20 px-3 py-1 text-xs"
          onClick={() => setOpen((v) => !v)}
        >
          {open ? '收起' : '航图轨'}
        </button>
      </header>
      {open && (
        <div className="fixed inset-x-0 top-[52px] z-40 grid grid-cols-4 gap-2 border-b border-foam/10 bg-kelp p-3 md:hidden">
          <NavItems onNavigate={() => setOpen(false)} />
        </div>
      )}

      <main className="w-full md:pl-[72px]">
        <Outlet />
      </main>
    </div>
  )
}
