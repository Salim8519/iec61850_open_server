import { defineConfig } from 'astro/config';

export default defineConfig({
  site: 'https://scadabench.com',
  output: 'static',
  trailingSlash: 'always',
  build: { inlineStylesheets: 'auto', assets: '_assets' },
  compressHTML: true,
  image: { responsiveStyles: false },
});
