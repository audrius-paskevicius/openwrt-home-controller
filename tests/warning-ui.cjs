// Real browser storage/navigation and HTTP responses; all controls are synthetic.
const assert=require('node:assert/strict'),fs=require('node:fs'),http=require('node:http');
const endpoint='http://127.0.0.1:'+(process.env.CHROME_PORT||9222);
const canonical=fs.readFileSync('DISCLAIMER.txt','utf8');
let warningText=canonical,unavailable=true;
const requests=[];
const state={csrf:'fixture',devices:[{id:'blind',name:'Test blind',state:{}}],groups:[],scheduler:{running:true,enabled_schedules:0,timezone:'UTC'}};
const server=http.createServer((req,res)=>{
 const path=new URL(req.url,'http://localhost').pathname;requests.push({path,method:req.method});
 res.setHeader('Cache-Control','no-store');
 if(path==='/disclaimer.txt'){res.writeHead(unavailable?503:200,{'Content-Type':'text/plain; charset=utf-8'});res.end(unavailable?'Unavailable':warningText);return;}
 if(path==='/api/state'){res.setHeader('Content-Type','application/json');res.end(JSON.stringify(state));return;}
 if(path==='/fixture'){res.setHeader('Content-Type','text/html');res.end('<!doctype html><body></body>');return;}
 const files={'/':'index.html','/app.js':'app.js','/style.css':'style.css','/manifest.json':'manifest.json','/icon.svg':'icon.svg','/icon-192.png':'icon-192.png','/icon-512.png':'icon-512.png','/apple-touch-icon.png':'apple-touch-icon.png'};
 const file=files[path];if(!file){res.writeHead(404);res.end();return;}
 const type=file.endsWith('.html')?'text/html':file.endsWith('.js')?'text/javascript':file.endsWith('.css')?'text/css':file.endsWith('.json')?'application/json':file.endsWith('.svg')?'image/svg+xml':'image/png';
 res.setHeader('Content-Type',type);res.end(fs.readFileSync('web/'+file));
});
async function browser(){
 const tab=await(await fetch(endpoint+'/json/new?about:blank',{method:'PUT'})).json(),ws=new WebSocket(tab.webSocketDebuggerUrl);
 await new Promise((resolve,reject)=>{ws.onopen=resolve;ws.onerror=reject;});let seq=0;const pending=new Map();
 ws.onmessage=event=>{const m=JSON.parse(event.data),p=pending.get(m.id);if(p){clearTimeout(p.timer);pending.delete(m.id);m.error?p.reject(Error(m.error.message)):p.resolve(m.result);}};
 const call=(method,params={})=>new Promise((resolve,reject)=>{const id=++seq,timer=setTimeout(()=>{pending.delete(id);reject(Error('Timeout: '+method));},15000);pending.set(id,{resolve,reject,timer});ws.send(JSON.stringify({id,method,params}));});
 const run=async expression=>{const r=await call('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true});if(r.exceptionDetails)throw Error(JSON.stringify(r.exceptionDetails));return r.result.value;};
 const wait=async expression=>{for(let i=0;i<100;i++){if(await run(expression))return;await new Promise(resolve=>setTimeout(resolve,50));}throw Error('Browser condition not reached: '+expression);};
 return {call,run,wait,close:async()=>{await fetch(endpoint+'/json/close/'+tab.id);ws.close();}};
}
(async()=>{
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));const origin='http://127.0.0.1:'+server.address().port,p=await browser();
 try{
  await p.call('Page.enable');await p.call('Page.bringToFront');await p.call('Page.navigate',{url:origin+'/'});
  await p.wait('document.querySelector("#warning button")?.textContent==="Retry"');
  assert.equal(requests.filter(r=>r.path.startsWith('/api/')).length,0);
  unavailable=false;await p.run('document.querySelector("#warning button").click()');
  await p.wait('document.querySelector("#warning button")?.textContent==="I understand — continue"');
  assert.deepEqual(await p.run('[...document.querySelectorAll("#warning h1,#warning p:not(.warning-note)")].map(x=>x.textContent)'),canonical.trim().split(/\n\s*\n/));
  await p.run('refresh()');assert.equal(requests.filter(r=>r.path.startsWith('/api/')).length,0);
  await p.run('document.querySelector("#warning button").click()');await p.wait('!!document.getElementById("scheduler")');
  assert.equal(await p.run('localStorage.getItem("home-controller-disclaimer")'),canonical);
  await p.call('Page.reload',{ignoreCache:true});await p.wait('!!document.getElementById("scheduler")');assert.equal(await p.run('document.getElementById("warning").hidden'),true);
  await p.run('document.getElementById("warning-link").click()');assert.equal(await p.run('document.querySelector("#warning button").textContent'),'Back');
  assert.equal(await p.run('document.getElementById("controls").hidden'),true);
  await p.run('document.querySelector("#warning button").click()');await p.wait('!document.getElementById("controls").hidden');
  warningText=canonical+'\nChanged warning fixture.\n';await p.call('Page.reload',{ignoreCache:true});await p.wait('document.querySelector("#warning button")?.textContent==="I understand — continue"');
  assert.equal(await p.run('document.getElementById("controls").hidden'),true);
  await p.run('document.querySelector("#warning button").click()');await p.wait('!!document.getElementById("scheduler")');
  assert.equal(await p.run('localStorage.getItem("home-controller-disclaimer")'),warningText);
  const blocked=await p.call('Page.addScriptToEvaluateOnNewDocument',{source:'Object.defineProperty(window,"localStorage",{configurable:true,get(){throw new DOMException("Blocked","SecurityError");}});'});
  await p.call('Page.reload',{ignoreCache:true});await p.wait('document.querySelector("#warning button")?.textContent==="I understand — continue"');
  await p.run('document.querySelector("#warning button").click()');await p.wait('!!document.getElementById("scheduler")');
  await p.call('Page.reload',{ignoreCache:true});await p.wait('document.querySelector("#warning button")?.textContent==="I understand — continue"');
  assert.ok(requests.every(r=>r.method==='GET'));
  console.log('PASS: control-page warning, load retry, exact-text persistence, changed text, read-again link, blocked storage; acknowledgement sends no control requests');
  await p.call('Page.removeScriptToEvaluateOnNewDocument',{identifier:blocked.identifier});await p.call('Page.navigate',{url:origin+'/fixture'});await p.wait('location.pathname==="/fixture"');
  warningText=canonical;
  await p.run(`window.E=(tag,attrs={},children=[])=>{const el=document.createElement(tag);for(const [key,value]of Object.entries(attrs)){if(typeof value==='function')el.addEventListener(key,value);else el.setAttribute(key,value);}for(const child of Array.isArray(children)?children:[children])el.append(child);return el;};window.ui={showModal:()=>{},hideModal:()=>{}};window.L={Class:{extend:x=>x},resource:()=>'/disclaimer.txt'};window.request={get:async url=>{const r=await fetch(url),text=await r.text();return {ok:r.ok,text:()=>text};}};window.makeWarning=()=>new Function('L','request','ui','E',${JSON.stringify(fs.readFileSync('luci/disclaimer.js','utf8'))})(L,request,ui,E);window.warning=makeWarning();window.text=await warning.load();window.renders=0;window.renderSettings=()=>{renders++;return E('div',{id:'settings'},'Settings');};localStorage.removeItem('home-controller-disclaimer');document.body.replaceChildren(warning.gate(text,renderSettings));` .replace('window.text=await warning.load();','window.text='+JSON.stringify(canonical)+';'));
  assert.equal(await p.run('renders'),0);
  assert.equal(await p.run('warning.load()'),canonical);
  await p.run('document.querySelector("button").click()');assert.equal(await p.run('renders'),1);assert.equal(await p.run('localStorage.getItem("home-controller-disclaimer")'),canonical);
  await p.run('warning=makeWarning();document.body.replaceChildren(warning.gate(text,renderSettings))');assert.equal(await p.run('!!document.getElementById("settings")'),true);
  await p.run('document.body.replaceChildren(warning.gate(text+"changed",renderSettings))');assert.equal(await p.run('document.querySelector("button").textContent'),'I understand — continue');
  await p.run('Object.defineProperty(window,"localStorage",{get(){throw Error("Blocked");}});warning=makeWarning();document.body.replaceChildren(warning.gate(text,renderSettings));document.querySelector("button").click();document.body.replaceChildren(warning.gate(text,renderSettings));');
  assert.equal(await p.run('!!document.getElementById("settings")'),true);
  console.log('PASS: LuCI exact-text acknowledgement, persistence across module loads, changed wording and in-memory fallback');
 }finally{await p.close();server.closeAllConnections();await new Promise(resolve=>server.close(resolve));}
})().catch(error=>{console.error(error);server.closeAllConnections();server.close();process.exitCode=1;});
