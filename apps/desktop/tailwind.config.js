/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        barq: {
          50:  "#f0f4ff",
          100: "#dde6ff",
          200: "#c3d0ff",
          300: "#9db1ff",
          400: "#7088ff",
          500: "#4a5eff",
          600: "#3340f5",
          700: "#2a32e0",
          800: "#252ab5",
          900: "#252a8f",
          950: "#161755",
        },
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "Fira Code", "monospace"],
      },
    },
  },
  plugins: [],
};
