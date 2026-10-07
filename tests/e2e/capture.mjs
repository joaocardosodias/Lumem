// Capture the persisted result after browser.mjs, waiting for data and animations.
import { chromium } from 'playwright';
import assert from 'node:assert/strict';
const browser = await chromium.launch({channel:process.env.E2E_BROWSER || 'msedge',headless:true});
try {
  const page=await browser.newPage({viewport:{width:1440,height:1000},locale:'pt-BR'});
  const base=process.env.E2E_URL || 'http://127.0.0.1:54173';
  await page.goto(base+'/dashboard');
  await page.locator('#email').fill('engineer@rf.test');
  await page.locator('#password').fill('BrowserRF!123');
  await page.getByRole('button',{name:'Logar',exact:true}).click();
  await page.locator('.overview-metric--completed strong').waitFor();
  assert.equal(await page.locator('.overview-metric--completed strong').innerText(),'1');
  await page.screenshot({path:'output/dashboard.png',fullPage:true,animations:'disabled'});
  await page.goto(base+'/predicoes');
  await page.locator('.predicoes-kpi-card.is-total strong').waitFor();
  assert.equal(await page.locator('.predicoes-kpi-card.is-total strong').innerText(),'528');
  await page.screenshot({path:'output/predicoes.png',fullPage:true,animations:'disabled'});
  await page.goto(base+'/manutencoes');
  await page.locator('.maintenance-record').first().waitFor();
  await page.screenshot({path:'output/manutencoes.png',fullPage:true,animations:'disabled'});
  await page.goto(base+'/inspecoes');
  await page.getByRole('button',{name:'Ver detalhes',exact:true}).first().click();
  const dialog=page.getByRole('dialog');
  assert.equal(await dialog.getByRole('button',{name:'Concluída',exact:true}).isDisabled(),true);
  await page.screenshot({path:'output/inspecao.png',fullPage:true,animations:'disabled'});
  console.log('Capturas finais obtidas após consulta dos registros persistidos.');
} finally { await browser.close(); }
