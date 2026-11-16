import { Events } from '@wailsio/runtime'
import { Accent } from '../../bindings/github.com/cn8001/reforgepulse/internal/app/themeservice'

// CSS can follow light and dark itself but cannot read the Windows accent
// colour, so it is fetched now and again whenever the theme changes.
export function followAccent(): void {
  const apply = () =>
    Accent()
      .then((c) => c && document.documentElement.style.setProperty('--accent', c))
      .catch(() => {}) // keep the default accent
  apply()
  Events.On('common:ThemeChanged', apply)
  window.addEventListener('focus', apply)
}
