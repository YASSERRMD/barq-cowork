/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        // Barq accent — a calm indigo-blue
        barq: {
          50:  "#eef2ff",
          100: "#e0e7ff",
          200: "#c7d2fe",
          300: "#a5b4fc",
          400: "#818cf8",
          500: "#6366f1",
          600: "#4f46e5",
          700: "#4338ca",
          800: "#3730a3",
          900: "#312e81",
          950: "#1e1b4b",
        },
        // Surface scale for layering
        surface: {
          0:  "#0a0a0f",  // deepest background
          1:  "#111118",  // app background
          2:  "#16161f",  // sidebar
          3:  "#1c1c27",  // cards, panels
          4:  "#22222f",  // hover states, nested cards
          5:  "#2a2a3a",  // borders, dividers
        },
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "-apple-system", "sans-serif"],
        mono: ["JetBrains Mono", "Fira Code", "Consolas", "monospace"],
      },
      fontSize: {
        "2xs": ["10px", { lineHeight: "14px" }],
      },
      borderRadius: {
        "sm": "4px",
        DEFAULT: "6px",
        "md": "8px",
        "lg": "10px",
        "xl": "12px",
      },
      boxShadow: {
        "panel": "0 0 0 1px rgba(255,255,255,0.06), 0 4px 16px rgba(0,0,0,0.4)",
        "card":  "0 0 0 1px rgba(255,255,255,0.04), 0 2px 8px rgba(0,0,0,0.3)",
        "sm":    "0 1px 3px rgba(0,0,0,0.5)",
      },
      transitionDuration: {
        DEFAULT: "150ms",
      },
    },
  },
  plugins: [],
};
