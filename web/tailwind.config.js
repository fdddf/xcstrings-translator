/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,ts,tsx,js,jsx}'
  ],
  theme: {
    extend: {
      fontFamily: {
        display: ['"Space Grotesk"', '"DM Sans"', 'system-ui', 'sans-serif'],
        body: ['"DM Sans"', '"Inter"', 'system-ui', 'sans-serif']
      },
      colors: {
        midnight: '#0b1021',
        ocean: '#1f4b99',
        mint: '#9ef4c5'
      },
      boxShadow: {
        neon: '0 10px 35px rgba(0, 0, 0, 0.25)',
        glow: '0 0 0 1px rgba(158, 244, 197, 0.3)'
      }
    }
  },
  plugins: []
}
