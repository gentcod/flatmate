/* eslint-disable react-refresh/only-export-components -- hook + banners live here by design */
import { useEffect, useState } from 'react'
import { useRegisterSW } from 'virtual:pwa-register/react'

/** Tracks navigator.onLine so the UI can say plainly when it is offline. */
export function useOnline(): boolean {
  const [online, setOnline] = useState(() =>
    typeof navigator === 'undefined' ? true : navigator.onLine,
  )
  useEffect(() => {
    const on = () => setOnline(true)
    const off = () => setOnline(false)
    window.addEventListener('online', on)
    window.addEventListener('offline', off)
    return () => {
      window.removeEventListener('online', on)
      window.removeEventListener('offline', off)
    }
  }, [])
  return online
}

/** Prompt banner when the service worker has fetched a newer build. */
export function UpdateBanner() {
  const {
    needRefresh: [needRefresh, setNeedRefresh],
    updateServiceWorker,
  } = useRegisterSW()
  if (!needRefresh) return null
  return (
    <div
      role="status"
      className="flex flex-wrap items-center justify-center gap-3 bg-eucalyptus px-4 py-2 text-center text-sm font-medium text-white"
    >
      <span>A new version of Flatmate is available.</span>
      <button
        type="button"
        onClick={() => void updateServiceWorker(true)}
        className="cursor-pointer rounded-md bg-canvas px-3 py-1 text-sm font-semibold text-eucalyptus hover:bg-parchment"
      >
        Update now
      </button>
      <button
        type="button"
        onClick={() => setNeedRefresh(false)}
        className="cursor-pointer rounded-md px-3 py-1 text-sm font-semibold text-[#aacec0] hover:bg-canopy"
      >
        Later
      </button>
    </div>
  )
}

/** Honest offline notice: cached pages render, but anything that saves needs a connection. */
export function OfflineBanner() {
  const online = useOnline()
  if (online) return null
  return (
    <div
      role="alert"
      className="bg-ink px-4 py-2 text-center text-sm font-medium text-canvas"
    >
      You&apos;re offline — saved pages still open, but changes need a connection.
    </div>
  )
}
