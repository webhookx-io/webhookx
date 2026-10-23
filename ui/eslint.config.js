import js from '@eslint/js'
import eslintConfigPrettier from 'eslint-config-prettier'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import globals from 'globals'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  {
    ignores: ['dist', 'node_modules', 'vite.config.js', 'vite.config.d.ts', '*.tsbuildinfo'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommendedTypeChecked,
  reactHooks.configs.flat.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      globals: globals.browser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      'react-refresh': reactRefresh,
    },
    rules: {
      'react-refresh/only-export-components': 'off',
      'react-hooks/incompatible-library': 'off',
      'react-hooks/set-state-in-effect': 'off',
      '@typescript-eslint/consistent-type-imports': ['error', { fixStyle: 'inline-type-imports' }],
      '@typescript-eslint/no-misused-promises': [
        'error',
        { checksVoidReturn: { attributes: false } },
      ],
    },
  },
  {
    files: ['**/*.config.{js,ts}'],
    ...tseslint.configs.disableTypeChecked,
    languageOptions: {
      ...tseslint.configs.disableTypeChecked.languageOptions,
      globals: globals.node,
    },
  },
  eslintConfigPrettier,
)
