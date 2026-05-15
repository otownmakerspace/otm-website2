/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './layouts/**/*.html',
    './content/**/*.{html,md}',
    './themes/hugo-up-business-main/layouts/**/*.html',
    './themes/hugo-up-business-main/assets/js/**/*.js',
    '../api/internal/**/*.html',
  ],
  safelist: [
    'bg-primary-2/50',
    'bg-primary-2/0',
    'group-hover:bg-primary-2/0',
    'bg-accent/15',
    'bg-accent/20',
    'bg-secondary/70',
    // zxcvbn strength-meter colors — used as JS string literals so
    // Tailwind's content scanner can't see them in markup.
    'bg-red-500',
    'bg-orange-500',
    'bg-yellow-500',
    'bg-lime-500',
    'bg-green-500',
  ],
  theme: {
    extend: {
      colors: {
        'accent': 'rgb(var(--c-accent) / <alpha-value>)',
        'accent-2': 'rgb(var(--c-accent-2) / <alpha-value>)',
        'primary': 'rgb(var(--c-primary) / <alpha-value>)',
        'primary-2': 'rgb(var(--c-primary-2) / <alpha-value>)',
        'secondary': 'rgb(var(--c-secondary) / <alpha-value>)',
        'secondary-2': 'rgb(var(--c-secondary-2) / <alpha-value>)',
        'gray-light': 'rgb(var(--c-gray-light) / <alpha-value>)',
        'gray-dark': 'rgb(var(--c-gray-dark) / <alpha-value>)',
        'tertiary': 'rgb(var(--c-tertiary) / <alpha-value>)',
        'surface': 'rgb(var(--c-surface) / <alpha-value>)',
      },
      fontFamily: {
        'primary': ['Roboto', 'sans-serif'],
        'secondary': ['Space Grotesk', 'sans-serif'],
      },
      borderRadius: {
        'DEFAULT': '10px',
        '4xl': '2rem',
      },
      spacing: {
        'section': '100px',
        'section-mobile': '50px',
        'navbar': '80px',
      },
    },
  },
  plugins: [],
}
