// Run against a disposable headless Chrome on localhost:9222. All API/RPC calls
// are synthetic: this test never connects to a router or transmits radio commands.
const assert=require('node:assert/strict'),fs=require('node:fs');
const endpoint='http://127.0.0.1:'+(process.env.CHROME_PORT||9222);
async function connection(){
 const tab=await (await fetch(endpoint+'/json/new?about:blank',{method:'PUT'})).json();
 const ws=new WebSocket(tab.webSocketDebuggerUrl);await new Promise((resolve,reject)=>{ws.onopen=resolve;ws.onerror=reject;});
 let id=0;const pending=new Map();
 ws.onmessage=e=>{const m=JSON.parse(e.data);if(pending.has(m.id)){const {resolve,reject,timer}=pending.get(m.id);clearTimeout(timer);pending.delete(m.id);m.error?reject(Error(m.error.message)):resolve(m.result);}};
 const call=(method,params={})=>new Promise((resolve,reject)=>{const n=++id,timer=setTimeout(()=>{pending.delete(n);reject(Error('CDP timeout: '+method));},15000);pending.set(n,{resolve,reject,timer});ws.send(JSON.stringify({id:n,method,params}));});
 const run=async expression=>{const r=await call('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true});if(r.exceptionDetails)throw Error(JSON.stringify(r.exceptionDetails));return r.result.value;};
 return {call,run,close:async()=>{ws.close();await fetch(endpoint+'/json/close/'+tab.id);}};
}
const fixture={csrf:'synthetic-token',devices:['Kitchen','Kids room','Bedroom','Office'].map((name,i)=>({id:'device'+i,name,state:{}})),groups:[{id:'all',name:'All blinds',devices:['device0','device1','device2','device3']}],scheduler:{running:true,enabled_schedules:1,timezone:'Europe/Stockholm',next:{target:'All blinds',action:'up',at:'2026-11-09T07:00:00Z'}}};
(async()=>{
 const page=await connection();try{
 // The production page loads external assets under CSP. The isolated fixture
 // injects the same assets; live deployment checks retain the production CSP.
 await page.call('Page.setBypassCSP',{enabled:true});
 let html=fs.readFileSync('web/index.html','utf8').replace(/<script\b[^>]*>[\s\S]*?<\/script>/g,'').replace(/<link\b[^>]*>/g,'');
 await page.call('Page.setDocumentContent',{frameId:(await page.call('Page.getFrameTree')).frameTree.frame.id,html});
 await page.run(`document.head.append(Object.assign(document.createElement('style'),{textContent:${JSON.stringify(fs.readFileSync('web/style.css','utf8'))}}));window.fixture=${JSON.stringify(fixture)};window.requests=[];window.failNext=false;window.fetch=async(path,options)=>{const body=options.body?JSON.parse(options.body):null;requests.push({path,body});if(failNext){failNext=false;return {ok:false,status:500,json:async()=>({error:'Synthetic failure'})};}if(path.endsWith('/scheduler'))fixture.scheduler.running=body.action==='start';return {ok:true,status:200,json:async()=>path.endsWith('/scheduler')?fixture.scheduler:fixture};};`);
 await page.run(fs.readFileSync('web/app.js','utf8'));
 await page.run('refresh()');
 assert.deepEqual(await page.run('[...document.querySelectorAll("article h2")].map(x=>x.textContent)'),['Kitchen','Kids room','Bedroom','Office','Scheduler','All blinds']);
 assert.equal(await page.run('document.querySelectorAll("header").length'),0);
 for(const width of [320,390,1280,1920]){
  await page.call('Emulation.setDeviceMetricsOverride',{width,height:844,deviceScaleFactor:1,mobile:width<650});
  const layout=await page.run(`({width:innerWidth,scroll:document.documentElement.scrollWidth,bottom:document.querySelector('footer').getBoundingClientRect().bottom,columns:getComputedStyle(document.getElementById('controls')).gridTemplateColumns.split(' ').filter(v=>parseFloat(v)>0).length,buttons:[...document.querySelectorAll('.card:first-child button')].map(b=>{const r=b.getBoundingClientRect();return {text:b.textContent,x:r.x,y:r.y,height:r.height,width:r.width};})})`);
  assert.ok(layout.scroll<=layout.width,JSON.stringify(layout));assert.deepEqual(layout.buttons.map(b=>b.text),['Up','Stop','Down']);
  assert.ok(layout.buttons.every(b=>b.height>=44&&b.width>=100),JSON.stringify(layout));assert.equal(layout.buttons[0].x,layout.buttons[2].x);assert.ok(layout.buttons[0].y<layout.buttons[1].y&&layout.buttons[1].y<layout.buttons[2].y);
  assert.equal(layout.buttons[1].y-layout.buttons[0].y-layout.buttons[0].height,8);
  if(width===390)assert.ok(layout.bottom<775,JSON.stringify(layout));
  if(width>=1280)assert.equal(layout.columns,6);
  if(process.env.UI_SCREENSHOTS){fs.mkdirSync('.private',{recursive:true});fs.writeFileSync('.private/redesign-'+width+'.png',Buffer.from((await page.call('Page.captureScreenshot',{format:'png',captureBeyondViewport:true})).data,'base64'));}
 }
 await page.run('fixture.devices.push(...[4,5,6,7].map(i=>({id:"device"+i,name:"Extra "+i,state:{}})));refresh()');
 assert.equal(await page.run('getComputedStyle(document.getElementById("controls")).gridTemplateColumns.split(" ").filter(v=>parseFloat(v)>0).length'),10);
 assert.equal(await page.run('document.querySelector("#scheduler .symbol").dataset.symbol'),'clock');
 await page.run('fixture.devices[0].icon="lamp";fixture.devices[1].icon="none";fixture.groups[0].icon="plug";fixture.scheduler_icon="fan";refresh()');
 assert.equal(await page.run('document.querySelector("[data-id=device0] .symbol").dataset.symbol'),'lamp');
 assert.equal(await page.run('document.querySelector("[data-id=device1] .symbol")'),null);
 assert.equal(await page.run('document.querySelector("[data-id=all] .symbol").dataset.symbol'),'plug');
 assert.equal(await page.run('document.querySelector("#scheduler .symbol").dataset.symbol'),'fan');
 await page.run('document.querySelector("#scheduler .stop").click()');
 assert.equal(await page.run('document.querySelector("#scheduler .status").textContent'),'Stopped');
 assert.equal(await page.run('document.querySelector("#scheduler .start").disabled'),false);
 await page.run('document.querySelector(".card .up").click()');assert.deepEqual(await page.run('requests.at(-1).body'),{target:'device0',action:'up'});
 await page.run('failNext=true;document.querySelector("#scheduler .start").click()');
 assert.equal(await page.run('document.querySelector("#scheduler .status").textContent'),'Synthetic failure');
 await page.run('document.querySelector("#scheduler .start").click()');
 assert.equal(await page.run('document.querySelector("#scheduler .status").textContent'),'Running');
 console.log('PASS: compact phone layout, 46px controls with 8px gaps, six desktop columns, configurable symbols/None, scheduler controls (synthetic API)');
 }finally{await page.close();}
 const admin=await connection();try{
 const config=JSON.parse(fs.readFileSync('files/config.json','utf8'));config.devices=[{id:'sample',name:'Test blind',radio_id:1,house:1234,unit:0}];config.groups=[{id:'all',name:'All blinds',devices:['sample']}];
 await admin.run(`window.saved=[];window.failSave=false;window.data=${JSON.stringify({config,account:'household',controllers:[],networks:['lan'],sunwait:true,status:{scheduler:fixture.scheduler}})};
 window.E=(tag,attrs={},children=[])=>{const e=document.createElement(tag);for(const [key,value] of Object.entries(attrs)){if(typeof value==='function')e.addEventListener(key,value);else if(key==='class')e.className=value;else e.setAttribute(key,value);}for(const c of Array.isArray(children)?children:[children])e.append(c);return e;};

 window.rpc={declare:spec=>(...args)=>{if(spec.method==='save'){if(failSave){failSave=false;return Promise.resolve({error:'Synthetic save failure'});}saved.push(JSON.parse(args[0]));return Promise.resolve({ok:true});}return Promise.resolve(data);}};window.poll={add:()=>{}};window.view={extend:x=>x};`);
 await admin.run(fs.readFileSync('tests/luci-fixture.js','utf8'));
 await admin.run(`document.body.append(new Function('view','rpc','ui','E','poll',${JSON.stringify(fs.readFileSync('luci/home-controller.js','utf8'))})(view,rpc,ui,E,poll).render(data));window.press=text=>[...document.querySelectorAll('button,.cbi-tabmenu a')].find(x=>x.textContent===text).click();window.input=label=>document.getElementById([...document.querySelectorAll('#modal label.cbi-value-title')].find(x=>x.textContent===label).htmlFor);`);
 assert.equal(await admin.run('document.querySelectorAll(".cbi-tabmenu a").length'),5);
 await admin.run('press("Devices & groups")');assert.equal(await admin.run('document.getElementById("hc-panel-devices").hidden'),false);
 await admin.run('press("Edit")');assert.equal(await admin.run('document.querySelector("#modal details").open'),false);await admin.run('press("Close")');
 await admin.run('press("Add device");input("Name").value="Second blind";input("House code").value="1235";input("Unit").value="1";input("Symbol").value="lamp";press("Save changes")');
 assert.equal(await admin.run('saved.at(-1).devices.length'),2);assert.match(await admin.run('saved.at(-1).devices[1].id'),/^device_/);
 assert.equal(await admin.run('saved.at(-1).devices[1].icon'),'lamp');
 await admin.run('press("Save changes")');assert.equal(await admin.run('saved.at(-1).devices.length'),2);
 await admin.run('press("Close");document.querySelectorAll("#hc-panel-devices table")[1].querySelector("button").click();input("Symbol").value="none";press("Save changes")');
 assert.equal(await admin.run('saved.at(-1).groups[0].icon'),'none');
 await admin.run('press("Close");press("Schedules");press("Add schedule");press("Save changes")');
 const schedule=await admin.run('saved.at(-1).schedules[0]');assert.equal(schedule.enabled,false);assert.equal(schedule.target,'sample');assert.deepEqual(schedule.days,[1,2,3,4,5]);assert.equal(schedule.time,'08:00');
 await admin.run('press("Close");window.toggle=document.querySelector("#hc-panel-schedules input[type=checkbox]");toggle.checked=true;toggle.dispatchEvent(new Event("change"))');
 assert.equal(await admin.run('saved.at(-1).schedules[0].enabled'),true);
 assert.equal(await admin.run('document.querySelector("#hc-panel-schedules .hc-row").querySelector("[role=status]").textContent'),'Saved.');
 await admin.run('failSave=true;toggle=document.querySelector("#hc-panel-schedules input[type=checkbox]");toggle.checked=false;toggle.dispatchEvent(new Event("change"))');
 assert.equal(await admin.run('toggle.checked'),true);assert.equal(await admin.run('document.querySelector("#hc-panel-schedules .hc-row").querySelector("[role=status]").textContent'),'Synthetic save failure');
 await admin.run('[...document.querySelectorAll("#hc-panel-schedules button")].find(x=>x.textContent==="Delete").click();[...document.querySelectorAll("#modal button")].find(x=>x.textContent==="Delete").click()');
 assert.equal(await admin.run('saved.at(-1).schedules.length'),0);assert.equal(await admin.run('[...document.querySelectorAll("#modal button")].map(x=>x.textContent).join()'),'Close');
 await admin.run('press("Close");press("Household access");document.getElementById([...document.querySelectorAll("label")].find(x=>x.textContent==="Scheduler symbol").htmlFor).value="none";press("Save symbol")');
 assert.equal(await admin.run('saved.at(-1).scheduler_icon'),'none');
 assert.equal(await admin.run('document.querySelectorAll(".hc-admin link[rel=stylesheet]").length'),0);
 console.log('PASS: LuCI tabs, device auto-ID and repeated save, disabled schedule defaults, weekday/time controls, inline success/error with checkbox rollback (synthetic RPC)');
 }finally{await admin.close();}
})().catch(e=>{console.error(e);process.exitCode=1;});
