import { create } from 'zustand'
import { authService } from '@/services/auth'

interface AuthState {
  isAuthenticated: boolean
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (name: string, email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  checkAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: authService.isAuthenticated(),
  isLoading: false,

  login: async (email: string, password: string) => {
    set({ isLoading: true })
    try {
      const response = await authService.login({ email, password })
      authService.setToken(response.token)
      set({ isAuthenticated: true })
    } finally {
      set({ isLoading: false })
    }
  },

  register: async (name: string, email: string, password: string) => {
    set({ isLoading: true })
    try {
      const response = await authService.register({ name, email, password })
      authService.setToken(response.token)
      set({ isAuthenticated: true })
    } finally {
      set({ isLoading: false })
    }
  },

  logout: async () => {
    set({ isLoading: true })
    try {
      await authService.logout()
    } finally {
      set({ isAuthenticated: false, isLoading: false })
    }
  },

  checkAuth: () => {
    set({ isAuthenticated: authService.isAuthenticated() })
  },
}))
