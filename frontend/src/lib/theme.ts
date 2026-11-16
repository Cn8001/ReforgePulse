import { ref, watch } from 'vue'
import { Events } from '@wailsio/runtime'
import { Accent } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/themeservice'

export type Theme = 'dark' | 'light'

// Storage can be unavailable; the theme then falls back to dark each start.
function stored(): Theme {
  try {
    return localStorage.getItem('theme') === 'light' ? 'light' : 'dark'
  } catch {
    return 'dark'
  }
}

export const theme = ref<Theme>(stored())

watch(
  theme,
  (t) => {
    document.documentElement.dataset.theme = t
    try {
      localStorage.setItem('theme', t)
    } catch {}
  },
  { immediate: true },
)

// CSS cannot read the Windows accent colour, so it is fetched now and again
// whenever Windows reports a theme change or the window regains focus.
export function followAccent(): void {
  const apply = () =>
    Accent()
      .then((c) => c && document.documentElement.style.setProperty('--accent', c))
      .catch(() => {})
  apply()
  Events.On('common:ThemeChanged', apply)
  window.addEventListener('focus', apply)
}
