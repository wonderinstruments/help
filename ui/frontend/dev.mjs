import * as esbuild from 'esbuild';
import { cpSync, mkdirSync } from 'fs';

mkdirSync('dist/assets', { recursive: true });
cpSync('style.css', 'dist/assets/style.css');
cpSync('index.html', 'dist/index.html');

const ctx = await esbuild.context({
  entryPoints: ['app.js'],
  bundle: true,
  outfile: 'dist/assets/app.js',
  format: 'esm',
  sourcemap: true,
});

await ctx.serve({ port: 34116, servedir: 'dist' });
console.log('Dev server running on http://localhost:34116');
