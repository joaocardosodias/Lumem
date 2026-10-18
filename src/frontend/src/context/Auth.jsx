import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { api, getToken, setToken, setUnauthorizedHandler, unwrapUser } from '../lib/api'
import { invalidateApplicationCache } from '../lib/appPrefetch'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [ready, setReady] = useState(false)

  function logout() {
    setToken(null)
    setUser(null)
    invalidateApplicationCache()
  }

  useEffect(() => {
    setUnauthorizedHandler(logout)
    return () => setUnauthorizedHandler(null)
  }, [])

  useEffect(() => {
    let cancelled = false

    async function boot() {
      if (!getToken()) {
        if (!cancelled) setReady(true)
        return
      }

      try {
        const profile = unwrapUser(await api.me())
        if (!cancelled) setUser(profile)
      } catch {
        setToken(null)
        if (!cancelled) setUser(null)
      } finally {
        if (!cancelled) setReady(true)
      }
    }

    boot()
    return () => {
      cancelled = true
    }
  }, [])

  async function authenticate(email, password) {
    const output = await api.login(email, password)
    const token = output?.token || output?.access_token
    if (!token) {
      throw new Error('A API não devolveu token.')
    }

    setToken(token)

    let profile = unwrapUser(output)
    if (!profile.email && !output?.user) {
      try {
        profile = unwrapUser(await api.me())
      } catch {
        profile = unwrapUser(output)
      }
    }

    return profile
  }

  function acceptUser(profile) {
    setUser(profile)
  }

  const value = useMemo(
    () => ({ user, ready, authenticate, acceptUser, logout }),
    [user, ready],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) {
    throw new Error('useAuth precisa do AuthProvider')
  }
  return value
}
