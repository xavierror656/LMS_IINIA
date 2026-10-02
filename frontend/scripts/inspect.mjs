import {chromium} from '@playwright/test';
import {mkdir,writeFile} from 'node:fs/promises';
const browser=await chromium.launch();const page=await browser.newPage({viewport:{width:1280,height:1000},reducedMotion:'reduce'});
const errors=[];page.on('pageerror',e=>errors.push(e.message));
await page.goto('http://localhost:4321/login');await page.getByLabel('Tu usuario').fill('luna');await page.getByLabel('Tu contraseña').fill(process.env.E2E_STUDENT_PASSWORD);await page.getByRole('button',{name:'Entrar a mi aventura'}).click();await page.waitForURL('**/courses');await page.getByLabel('Tu progreso').getByText('Luna',{exact:true}).waitFor();await page.evaluate(()=>document.fonts.ready);
await mkdir('../docs/screenshots',{recursive:true});await page.screenshot({path:'../docs/screenshots/courses-desktop.png',fullPage:true});await page.setViewportSize({width:820,height:1180});await page.screenshot({path:'../docs/screenshots/courses-tablet.png',fullPage:true});
const timing=await page.evaluate(async()=>{const result={};for(const path of ['/api/v1/courses','/api/v1/me/progress']){const samples=[];for(let i=0;i<20;i++){const start=performance.now();const r=await fetch(path);await r.text();if(!r.ok)throw Error('measurement request failed');samples.push(performance.now()-start)}samples.sort((a,b)=>a-b);result[path]={n:samples.length,medianMs:samples[10],p95Ms:samples[18]}}return result});
const initialJavaScript={};
const cookies=await page.context().cookies();
for(const path of ['/', '/courses','/lessons/1','/lessons/2','/lessons/3']){
 const context=await browser.newContext({reducedMotion:'reduce'});await context.addCookies(cookies);const sample=await context.newPage();await sample.goto('http://localhost:4321'+path);await sample.waitForLoadState('networkidle');
 initialJavaScript[path]=await sample.evaluate(()=>{const resources=performance.getEntriesByType('resource').filter(r=>new URL(r.name).pathname.endsWith('.js'));return {decodedBytes:resources.reduce((n,r)=>n+r.decodedBodySize,0),files:resources.map(r=>new URL(r.name).pathname),inlineBytes:[...document.scripts].filter(s=>!s.src).reduce((n,s)=>n+new TextEncoder().encode(s.textContent??'').length,0)}});await context.close();
}
await writeFile('../docs/browser-measurements.json',JSON.stringify({environment:process.env.INSPECT_MODE??'development',method:'20 warmed sequential requests via localhost proxy; Chromium 820x1180; initial JS from Resource Timing in fresh browser contexts, uncompressed bytes, H5P disabled',timing,initialJavaScript,pageErrors:errors},null,2));await browser.close();
