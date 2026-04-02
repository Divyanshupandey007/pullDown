/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    "./src/**/*.{html,ts}",
  ],
  theme: {
    extend: {
      colors: {
        "primary": "var(--accent-color)",
        "primary-container": "var(--glow-color)",
        "surface": "#09090b",
        "surface-container": "#121214",
        "surface-container-low": "#0e0e10",
        "surface-container-high": "#1a1a1c",
        "surface-container-highest": "#222224",
        "surface-bright": "#2c2c2e",
        "surface-variant": "#222224",
        "on-surface": "#e5e1e4",
        "on-surface-variant": "#a1a1aa",
        "outline": "#767575",
        "outline-variant": "rgba(255,255,255,0.1)",
        "background": "#09090b",
        "on-background": "#e5e1e4",
        "secondary": "#ff51fa",
        "secondary-container": "#a900a9",
        "tertiary": "#4ade80",
        "error": "#ff716c",
        "error-container": "#9f0519",
        "on-primary": "#000000",
        "on-primary-container": "#e5e1e4",
        "on-error": "#490006",
      },
      fontFamily: {
        "headline": ["Outfit", "sans-serif"],
        "body": ["Figtree", "sans-serif"],
        "label": ["Outfit", "sans-serif"]
      },
      borderRadius: {
        "DEFAULT": "0.375rem",
        "lg": "0.75rem",
        "xl": "1rem",
        "2xl": "1.5rem",
        "full": "9999px"
      },
    },
  },
  plugins: [],
}
