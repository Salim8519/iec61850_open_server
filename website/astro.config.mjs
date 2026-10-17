import { defineConfig } from 'astro/config';

export default defineConfig({
  site: 'https://scadabench.com',
  output: 'static',
  trailingSlash: 'always',
  build: { inlineStylesheets: 'auto', assets: '_assets' },
  compressHTML: true,
  // A page starts loading when the pointer rests on its link (touch start on phones), so
  // moving between pages feels instant without spending mobile data on links nobody opens.
  prefetch: { prefetchAll: true, defaultStrategy: 'hover' },
  image: { responsiveStyles: false },
});
