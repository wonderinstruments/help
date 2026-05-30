import * as esbuild from 'esbuild';
import { cpSync, mkdirSync } from 'fs';

mkdirSync('dist/assets', { recursive: true });

await esbuild.build({
  entryPoints: ['app.js'],
  bundle: true,
  minify: true,
  outfile: 'dist/assets/app.js',
  format: 'esm',
});

cpSync('style.css', 'dist/assets/style.css');
cpSync('index.html', 'dist/index.html');
