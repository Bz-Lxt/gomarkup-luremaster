import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '../components/ui/Button'
import { useToast } from '../context/ToastContext'

export function PlaceholderPage({ title }: { title: string }) {
  const { toast } = useToast()
  useEffect(() => {
    toast('即将开放', 'info')
  }, [toast])

  return (
    <div className="chart-grain flex min-h-screen w-full flex-col items-center justify-center px-6 text-center">
      <p className="font-mono text-[10px] tracking-[0.28em] text-mute">RESTRICTED</p>
      <h1 className="mt-2 font-display text-5xl italic">{title}</h1>
      <p className="mt-3 text-mute">舱门尚未打开，先回海图室。</p>
      <Link to="/" className="mt-6">
        <Button variant="ghost">返回港口</Button>
      </Link>
    </div>
  )
}
