import { chromium } from 'playwright';
import XLSX from 'xlsx';
import assert from 'node:assert/strict';
import { createHmac } from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

const base = process.env.E2E_URL || 'http://127.0.0.1:54173';
const output = path.resolve('output');
fs.mkdirSync(output, { recursive: true });
const results = { browser: 'Microsoft Edge / Chromium', environment: 'isolated Docker compose', simulatedServices: [], storage: 'real local filesystem shared by API and River worker', cases: [], requests: [], pageErrors: [] };
const browser = await chromium.launch({ channel: process.env.E2E_BROWSER || 'msedge', headless: true });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'pt-BR', timezoneId: 'America/Sao_Paulo' });
const page = await context.newPage();
page.setDefaultTimeout(30000);
page.on('pageerror', error => results.pageErrors.push(error.message));
page.on('response', response => {
  if (response.url().includes('/api/')) results.requests.push({ path: new URL(response.url()).pathname, method: response.request().method(), status: response.status() });
});
const active = () => page.locator('.app-page-cache:not([hidden])');
const filterSelect = label => active().locator('.filtros__campo').filter({ has: page.locator('span').filter({ hasText: new RegExp('^'+label+'$') }) }).locator('select');
async function step(name, action) {
  try { const details = await action(); results.cases.push({ name, status: 'passed', details }); console.log('PASS ' + name); }
  catch (error) { results.cases.push({ name, status: 'failed', error: error.message }); await page.screenshot({ path: path.join(output, 'failure.png'), fullPage: true, animations: 'disabled' }); fs.writeFileSync(path.join(output, 'failure.txt'), await page.locator('body').innerText()); throw error; }
}
async function count(locator, expected) {
  const end = Date.now() + 60000;
  while (Date.now() < end) { if (await locator.count() === expected) return; await page.waitForTimeout(200); }
  assert.equal(await locator.count(), expected);
}
async function navigate(route, heading) { await page.goto(base + route); await page.getByRole('heading', { name: heading, exact: true }).waitFor(); }
async function login(email) {
  await page.locator('#email').fill(email);
  await page.locator('#password').fill('BrowserRF!123');
  await page.getByRole('button', { name: 'Logar', exact: true }).click();
  await page.locator('.app-shell').waitFor();
}
async function api(route) {
  return page.evaluate(async route => {
    const response = await fetch('/api/v1' + route, { headers: { Authorization: 'Bearer ' + localStorage.getItem('motiva.token') } });
    return { status: response.status, body: await response.json() };
  }, route);
}
const list = body => Array.isArray(body) ? body : body.data || body.items || body.inspections || [];
const stamp = new Date(Date.now() - 86400000).toISOString().slice(0, 10);
function fixture(kind, format, offset) {
  const rows = kind === 'alarmes'
    ? [['tag','estacao','plataforma','porta','alarme','horario_alarme','tempo_normalizacao_seg'],
      ['SP04-6-PSD-FAL-P1-PP04','FAL','P1','4','OBSTACULO',stamp+' 10:0'+offset+':00','12'],
      ['SP04-6-PSD-PIN-P2-PP08','PIN','P2','8','FALHA',stamp+' 11:0'+offset+':00','20']]
    : [['ordem','tipo_de_ordem','local_de_instalacao','estacao','inicio_avaria_datetime','fim_avaria_datetime','duracao_avaria_horas','descricao','sintoma','causa','acao'],
      [String(9100+offset),'ZM01','SP04-6-PSD-FAL-P1-PP04','FAL',stamp+' 10:00:00',stamp+' 11:30:00','1.5','Reparo E2E','Ruido','Desgaste','Troca'],
      [String(9200+offset),'ZM01','SP04-6-PSD-PIN-P2-PP08','PIN',stamp+' 12:00:00',stamp+' 13:30:00','1.5','Reparo E2E','Ruido','Desgaste','Troca']];
  const destination = path.join(output, kind+'-'+offset+'.'+format);
  if (format === 'csv') fs.writeFileSync(destination, rows.map(r => r.join(',')).join('\n'));
  else { const book = XLSX.utils.book_new(); XLSX.utils.book_append_sheet(book, XLSX.utils.aoa_to_sheet(rows), 'Dados'); XLSX.writeFile(book, destination); }
  return destination;
}
try {
  await step('Login obrigatório e credenciais inválidas', async () => {
    await page.goto(base+'/manutencoes');
    await page.getByRole('heading', { name: 'Entrar', exact: true }).waitFor();
    await page.locator('#email').fill('operator@rf.test'); await page.locator('#password').fill('incorreta');
    const response = page.waitForResponse(r => r.url().endsWith('/auth/login') && r.status() === 401);
    await page.getByRole('button', { name: 'Logar', exact: true }).click(); await response;
    await page.locator('.login__error').waitFor(); assert.equal(await page.locator('.app-shell').count(), 0);
    return 'Página interna exige login; credenciais inválidas retornam 401 e exibem mensagem.';
  });
  await step('Login válido e persistência da sessão', async () => {
    await login('operator@rf.test'); await page.reload(); await page.locator('.app-shell').waitFor();
    assert.equal((await api('/auth/me')).status, 200);
    return 'Login pela interface e sessão mantida após recarregar.';
  });
  await step('Sessão expirada', async () => {
    const original = await page.evaluate(() => localStorage.getItem('motiva.token'));
    const payload = JSON.parse(Buffer.from(original.split('.')[1], 'base64url'));
    payload.exp = Math.floor(Date.now()/1000)-3600;
    const input = Buffer.from(JSON.stringify({ alg:'HS256',typ:'JWT' })).toString('base64url')+'.'+Buffer.from(JSON.stringify(payload)).toString('base64url');
    const expired = input+'.'+createHmac('sha256','rf-browser-isolated-test-signing-key').update(input).digest('base64url');
    await page.evaluate(token => { localStorage.setItem('motiva.token',token); sessionStorage.setItem('motiva.token',token); }, expired);
    const response = page.waitForResponse(r => r.url().endsWith('/auth/me') && r.status() === 401);
    await page.reload(); await response; await page.getByRole('heading',{name:'Entrar',exact:true}).waitFor();
    assert.equal(await page.evaluate(() => localStorage.getItem('motiva.token') || sessionStorage.getItem('motiva.token')), null);
    await login('operator@rf.test');
    return 'JWT corretamente assinado e já expirado: API 401, token removido e retorno ao login.';
  });
  await step('Upload inválido apresenta falha e permite nova tentativa', async () => {
    await navigate('/alarmes','Alarmes');
    await active().getByRole('button',{name:'Importar alarmes',exact:true}).click();
    const dialog=page.getByRole('dialog');
    await dialog.locator('input[type=file]').setInputFiles({name:'invalido.csv',mimeType:'text/csv',buffer:Buffer.from('coluna_invalida\nvalor\n')});
    const response=page.waitForResponse(r=>r.url().endsWith('/uploads/alarms')&&r.request().method()==='POST');
    await dialog.getByRole('button',{name:'Importar planilha',exact:true}).click();
    const accepted=await response; assert.equal(accepted.status(),202); const upload=await accepted.json();
    await dialog.getByText('A API recusou o processamento da planilha de alarmes.',{exact:true}).waitFor({timeout:90000});
    const final=(await api('/uploads/'+upload.upload_id)).body; assert.equal(final.status,'failed'); assert.equal(final.rows_inserted,0);
    await dialog.getByRole('button',{name:'Cancelar',exact:true}).click();
    return {status:final.status,inserted:final.rows_inserted};
  });
  for (const [kind, format, offset] of [['alarmes','csv',1],['alarmes','xlsx',2],['manutencoes','csv',1],['manutencoes','xlsx',2]]) {
    await step('Upload '+kind+' '+format+' pelo worker', async () => {
      await navigate('/'+kind, kind === 'alarmes' ? 'Alarmes' : 'Manutenções');
      await active().getByRole('button',{name:kind === 'alarmes' ? 'Importar alarmes' : 'Nova manutenção',exact:true}).click();
      const dialog=page.getByRole('dialog');
      if(kind !== 'alarmes') await dialog.getByRole('button',{name:'CSV / XLSX',exact:true}).click();
      await dialog.locator('input[type=file]').setInputFiles(fixture(kind,format,offset));
      const route=kind === 'alarmes' ? 'alarms' : 'maintenance';
      const response=page.waitForResponse(r=>r.url().endsWith('/uploads/'+route)&&r.request().method()==='POST');
      await dialog.getByRole('button',{name:kind === 'alarmes'?'Importar planilha':'Enviar planilha',exact:true}).click();
      const accepted=await response; assert.equal(accepted.status(),202); const upload=await accepted.json();
      await dialog.waitFor({state:'hidden',timeout:90000});
      const final=await api('/uploads/'+upload.upload_id); assert.equal(final.status,200); assert.equal(final.body.status,'completed'); assert.equal(final.body.rows_inserted,2);
      await count(active().locator(kind === 'alarmes'?'.card-alarme':'.maintenance-record'),offset*2);
      return { uploadId:upload.upload_id, ...final.body };
    });
  }
  await step('Cadastro manual e consulta após recarregar', async()=>{
    await active().getByRole('button',{name:'Nova manutenção',exact:true}).click();
    for(const [id,value] of Object.entries({'mnt-type':'ZM03','mnt-order':'9300','mnt-station':'FAL','mnt-location':'SP04-6-PSD-FAL-P1-PP04','mnt-start-date':stamp,'mnt-start-time':'14:00','mnt-end-date':stamp,'mnt-end-time':'15:30','mnt-symptom':'Inspeção E2E','mnt-desc':'Cadastro manual no navegador'})) await page.locator('#'+id).fill(value);
    const saved=page.waitForResponse(r=>r.url().endsWith('/maintenance')&&r.request().method()==='POST');
    await page.getByRole('dialog').getByRole('button',{name:'Cadastrar',exact:true}).click();
    assert.equal((await saved).status(),201); await page.getByRole('dialog').waitFor({state:'hidden'});
    await page.reload(); await count(active().locator('.maintenance-record'),5);
    const record=active().locator('.maintenance-record').filter({hasText:'9300'}); await record.locator('summary').click();
    assert.match(await record.innerText(),/Cadastro manual no navegador/); assert.match(await record.innerText(),/1,5 h/);
    return 'Registro persistido, campos e duração conferidos após recarregar; cinco manutenções no total.';
  });
  await step('Filtros combinados, vazio e limpeza', async()=>{
    await filterSelect('Estação').selectOption('FAL'); await count(active().locator('.maintenance-record'),3);
    await filterSelect('Tipo').selectOption('ZM03'); await count(active().locator('.maintenance-record'),1);
    await active().getByPlaceholder('Buscar por ordem, sintoma...').fill('inexistente-e2e'); await count(active().locator('.maintenance-record'),0);
    await active().getByRole('button',{name:'Limpar filtros',exact:true}).click(); await count(active().locator('.maintenance-record'),5);
    await page.screenshot({path:path.join(output,'manutencoes.png'),fullPage:true,animations:'disabled'});
    await navigate('/alarmes','Alarmes'); await filterSelect('Estação').selectOption('PIN'); await count(active().locator('.card-alarme'),2);
    return 'Manutenção: 5 → 3 → 1 → 0 → 5 registros; alarmes PIN: dois registros.';
  });
  await step('Predição com modelo real', async()=>{
    await context.clearCookies(); await page.evaluate(()=>{localStorage.clear();sessionStorage.clear();}); await page.reload(); await login('engineer@rf.test');
    await navigate('/predicoes','Predições');
    const response=page.waitForResponse(r=>r.url().endsWith('/predictions/trigger')&&r.request().method()==='POST',{timeout:120000});
    await active().getByRole('button',{name:'Executar análise',exact:true}).click();
    const predicted=await response; const body=await predicted.json(); assert.equal(predicted.status(),201,JSON.stringify(body));
    results.prediction=body; assert.ok(body.total_doors_evaluated>0); assert.ok(body.model_version);
    await page.locator('.predicoes-kpi-card.is-total strong').waitFor();
    assert.equal(Number(await page.locator('.predicoes-kpi-card.is-total strong').innerText()),body.total_doors_evaluated);
    await page.screenshot({path:path.join(output,'predicoes.png'),fullPage:true,animations:'disabled'});
    return {runId:body.id,model:body.model_version,evaluated:body.total_doors_evaluated,risk:body.total_doors_at_risk};
  });
  await step('Inspeções e atualização de status', async()=>{
    await navigate('/inspecoes','Inspeções');
    const response=await api('/inspections?limit=2000'); const inspections=list(response.body);
    const generated=inspections.filter(i=>i.prediction_id);
    let manualCreated=false;
    if (!generated.length) {
      assert.equal(results.prediction.total_doors_at_risk,0);
      await active().getByRole('button',{name:'Nova inspeção',exact:true}).click();
      const form=page.getByRole('dialog');
      await form.locator('select').nth(0).selectOption('FAL');
      await form.locator('select').nth(1).selectOption('P1');
      await form.locator('#insp-door').fill('4');
      await form.locator('#insp-notes').fill('Inspeção manual do fluxo E2E');
      const created=page.waitForResponse(r=>r.url().endsWith('/inspections')&&r.request().method()==='POST');
      await form.locator('button[type=submit]').click(); assert.equal((await created).status(),201);
      await form.waitFor({state:'hidden'}); manualCreated=true;
    }
    await active().getByRole('button',{name:'Ver detalhes',exact:true}).first().click();
    const dialog=page.getByRole('dialog');
    for(const name of ['Em andamento','Concluída']) {
      const updated=page.waitForResponse(r=>r.url().includes('/inspections/')&&r.url().endsWith('/status')&&r.request().method()!=='GET');
      await dialog.getByRole('button',{name,exact:true}).click(); const result=await updated; assert.equal(result.status(),200);
    }
    await page.screenshot({path:path.join(output,'inspecao.png'),fullPage:true,animations:'disabled'});
    await dialog.getByRole('button',{name:'Fechar',exact:true}).click();
    const after=list((await api('/inspections?status=completed&limit=2000')).body); assert.equal(after.length,1); assert.ok(after[0].completed_at);
    results.completedInspection=after[0];
    return {automatic:generated.length,manualCreated,completed:1,predictionId:after[0].prediction_id};
  });
  await step('Dashboard consistente com API e inspeções',async()=>{
    await navigate('/dashboard','Visão geral'); const response=await api('/dashboard/summary'); assert.equal(response.status,200);
    const summary=response.body.data||response.body; results.dashboard=summary;
    assert.equal(summary.doors_at_risk,results.prediction.total_doors_at_risk); assert.equal(summary.completed_inspections,1);
    for(const [selector,value] of [['risk',summary.doors_at_risk],['pending',summary.pending_inspections],['completed',summary.completed_inspections]]) {
      const element=page.locator('.overview-metric--'+selector+' strong'); await element.waitFor();
      assert.equal(Number((await element.innerText()).replace(/\D/g,'')),value);
    }
    await page.screenshot({path:path.join(output,'dashboard.png'),fullPage:true,animations:'disabled'});
    assert.equal(results.pageErrors.length,0,results.pageErrors.join('\n'));
    return summary;
  });
  await step('Logout bloqueia novo acesso à página interna', async()=>{
    await page.getByRole('button',{name:'Sair',exact:true}).click();
    await page.getByRole('heading',{name:'Entrar',exact:true}).waitFor();
    await page.goto(base+'/inspecoes'); await page.getByRole('heading',{name:'Entrar',exact:true}).waitFor();
    assert.equal(await page.evaluate(()=>localStorage.getItem('motiva.token')||sessionStorage.getItem('motiva.token')),null);
    return 'Sessão removida e página interna protegida após logout.';
  });
} catch(error) { console.error(error.message); process.exitCode=1; }
finally { fs.writeFileSync(path.join(output,'results.json'),JSON.stringify(results,null,2)); await browser.close(); }
