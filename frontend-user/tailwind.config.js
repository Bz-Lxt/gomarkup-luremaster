/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: '#07120F',
        kelp: '#0E2420',
        foam: '#E8DCC4',
        sonar: '#C6F04A',
        copper: '#D4783A',
        tide: '#5EC8D8',
        pressure: '#F2C14E',
        mute: '#8A9A8E',
      },
      fontFamily: {
        display: ['Fraunces', 'Georgia', 'serif'],
        body: ['Figtree', 'system-ui', 'sans-serif'],
        mono: ['IBM Plex Mono', 'ui-monospace', 'monospace'],
      },
      screens: {
        xs: '480px',
        md: '768px',
      },
      boxShadow: {
        rivet: 'inset 0 1px 0 rgba(232,220,196,0.08), 0 12px 40px rgba(0,0,0,0.35)',
      },
      keyframes: {
        fadeUp: {
          '0%': { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        copperSweep: {
          '0%': { transform: 'translateX(-120%)' },
          '100%': { transform: 'translateX(120%)' },
        },
      },
      animation: {
        fadeUp: 'fadeUp 420ms ease-out both',
        copperSweep: 'copperSweep 900ms ease-out 1',
      },
    },
  },
  plugins: [],
}
