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
 await page.call('Emulation.setLocaleOverride',{locale:'en-US'});
 await page.call('Emulation.setTimezoneOverride',{timezoneId:'Europe/Stockholm'});
 // The production page loads external assets under CSP. The isolated fixture
 // injects the same assets; live deployment checks retain the production CSP.
 await page.call('Page.setBypassCSP',{enabled:true});
 let html=fs.readFileSync('web/index.html','utf8').replace(/<script\b[^>]*>[\s\S]*?<\/script>/g,'').replace(/<link\b[^>]*>/g,'');
 await page.call('Page.setDocumentContent',{frameId:(await page.call('Page.getFrameTree')).frameTree.frame.id,html});
 await page.run(`document.head.append(Object.assign(document.createElement('style'),{textContent:${JSON.stringify(fs.readFileSync('web/style.css','utf8'))}}));window.fixture=${JSON.stringify(fixture)};window.requests=[];window.failNext=false;window.fetch=async(path,options)=>{if(path==='/disclaimer.txt')return {ok:true,text:async()=>${JSON.stringify(fs.readFileSync('DISCLAIMER.txt','utf8'))}};const body=options.body?JSON.parse(options.body):null;requests.push({path,body});if(failNext){failNext=false;return {ok:false,status:500,json:async()=>({error:'Synthetic failure'})};}if(path.endsWith('/scheduler'))fixture.scheduler.running=body.action==='start';return {ok:true,status:200,json:async()=>path.endsWith('/scheduler')?fixture.scheduler:fixture};};`);
 await page.run(fs.readFileSync('web/app.js','utf8'));
 await page.run('initialize()');
 await page.run('document.querySelector("#warning button").click()');
 await page.run('refresh()');
 await page.run('fixture.devices[0].state={action:"up",status:"Sent (position unknown)",at:Date.parse("2026-11-08T21:10:00Z")/1000};refresh()');
 assert.equal(await page.run('document.querySelector("[data-id=device0] .status").textContent'),'Up sent · 22:10');
 assert.match(await page.run('document.querySelector("#scheduler .next time").textContent'),/^2026-11-09 08:00$/);
 assert.equal(await page.run('document.querySelector("#scheduler time").dateTime'),'2026-11-09T07:00:00Z');
 assert.equal(await page.run('document.querySelector("#scheduler .next").textContent.includes("Europe/")'),false);
 for(const [instant,zone,want] of [
  ['2026-11-08T23:00:00Z','Europe/Stockholm','2026-11-09 00:00'],
  ['2026-10-25T00:30:00Z','Europe/Stockholm','2026-10-25 02:30'],
  ['2026-10-25T01:30:00Z','Europe/Stockholm','2026-10-25 02:30'],
  ['2026-11-08T01:00:00Z','America/New_York','2026-11-07 20:00'],
  ['2026-11-08T00:00:00Z','Asia/Kathmandu','2026-11-08 05:45'],
  ['2026-11-08T00:00:00Z','UTC','2026-11-08 00:00']
 ])assert.equal(await page.run(`dateText(${JSON.stringify(instant)},${JSON.stringify(zone)})`),want);
 await page.call('Emulation.setLocaleOverride',{locale:'ar-EG'});
 assert.equal(await page.run('dateText("2026-11-08T23:00:00Z",undefined,false)'),'00:00');
 assert.equal(await page.run('dateText("2026-11-08T21:10:00Z")'),'2026-11-08 22:10');
 await page.call('Emulation.setLocaleOverride',{locale:'en-US'});
 assert.deepEqual(await page.run('[...document.querySelectorAll("article h2")].map(x=>x.textContent)'),['Kitchen','Kids room','Bedroom','Office','Scheduler','All blinds']);
 assert.equal(await page.run('document.querySelectorAll("header").length'),0);
 for(const width of [320,390,1280,1920]){
  await page.call('Emulation.setDeviceMetricsOverride',{width,height:844,deviceScaleFactor:1,mobile:width<650});
  const layout=await page.run(`({width:innerWidth,scroll:document.documentElement.scrollWidth,bottom:document.querySelector('footer').getBoundingClientRect().bottom,columns:getComputedStyle(document.getElementById('controls')).gridTemplateColumns.split(' ').filter(v=>parseFloat(v)>0).length,buttons:[...document.querySelectorAll('.card:first-child button')].map(b=>{const r=b.getBoundingClientRect();return {text:b.textContent,x:r.x,y:r.y,height:r.height,width:r.width};})})`);
  assert.ok(layout.scroll<=layout.width,JSON.stringify(layout));assert.equal(await page.run('getComputedStyle(document.querySelector("#scheduler time")).whiteSpace'),'nowrap');assert.equal(await page.run('document.querySelector("#scheduler time").scrollWidth<=document.querySelector("#scheduler .next").clientWidth'),true);assert.deepEqual(layout.buttons.map(b=>b.text),['Up','Stop','Down']);
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
 await page.run('fixture.control_order=["group:all","scheduler","device:device0"];refresh()');
 assert.deepEqual(await page.run('[...document.querySelectorAll("article h2")].slice(0,3).map(x=>x.textContent)'),['All blinds','Scheduler','Kitchen']);
 // Group feedback belongs only to group requests. Queue stages and repeated
 // polls leave existing text alone instead of flashing transient messages.
 assert.equal(await page.run('document.querySelector("[data-id=all] .status").textContent'),'');
 await page.run('fixture.group_states={all:{action:"up",status:"Complete",at:Date.parse("2026-11-08T21:10:00Z")/1000,total:4,sent:4,failed:0,cancelled:0,pending:0}};refresh()');
 assert.equal(await page.run('document.querySelector("[data-id=all] .status").textContent'),'Up sent to all · 22:10');
 await page.run('fixture.devices[0].state={action:"down",status:"radio command failed",at:123};refresh()');
 assert.equal(await page.run('document.querySelector("[data-id=all] .status").textContent'),'Up sent to all · 22:10');
 assert.equal(await page.run('document.querySelector("[data-id=all]").classList.contains("error")'),false);
 await page.run('window.statusMutations=0;window.statusObserver=new MutationObserver(m=>statusMutations+=m.length);statusObserver.observe(document.querySelector("[data-id=all] .status"),{childList:true,characterData:true,subtree:true});fixture.group_states.all.status="Queued";fixture.group_states.all.pending=4;document.querySelector("[data-id=all] .down").click()');
 await page.run('refresh()');await page.run('refresh()');
 assert.equal(await page.run('document.querySelector("[data-id=all] .status").textContent'),'Up sent to all · 22:10');
 assert.equal(await page.run('statusMutations'),0);
 await page.run('fixture.group_states.all={action:"down",status:"Complete",at:Date.parse("2026-11-08T21:11:00Z")/1000,total:4,sent:3,failed:1,pending:0};refresh()');
 assert.equal(await page.run('document.querySelector("[data-id=all] .status").textContent'),'Down sent to 3/4 · 22:11');
 assert.equal(await page.run('document.querySelector("[data-id=all]").classList.contains("error")'),true);
 await page.run('statusObserver.disconnect()');
 await page.run('fixture.devices.push({id:"lamp",name:"Test lamp",type:"switch",state:{}});fixture.groups.push({id:"lights",name:"Lights",type:"switch",devices:["lamp"]});fixture.scheduler.next={target:"Lights",action:"off",at:"2026-11-09T21:00:00Z"};fixture.scheduler.enabled_schedules=1;refresh()');
 assert.deepEqual(await page.run('[...document.querySelectorAll("[data-id=lamp] button")].map(b=>b.textContent)'),['On','Off']);
 assert.deepEqual(await page.run('[...document.querySelectorAll("[data-id=lights] button")].map(b=>b.textContent)'),['On','Off']);
 assert.equal(await page.run('document.querySelector("[data-id=lamp] .status").textContent'),'State unknown');
 assert.equal(await page.run('document.querySelector("[data-id=lamp] .symbol").dataset.symbol'),'plug');
 assert.match(await page.run('document.querySelector("#scheduler .next-event").textContent'),/Lights Off/);
 await page.run('document.querySelector("[data-id=lamp] .on").click()');assert.deepEqual(await page.run('requests.at(-1).body'),{target:'lamp',action:'on'});
 await page.run('document.querySelector("[data-id=lights] .off").click()');assert.deepEqual(await page.run('requests.at(-1).body'),{target:'lights',action:'off'});
 assert.equal(await page.run('document.querySelectorAll("#scheduler").length'),1);
 await page.call('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:true});
 assert.equal(await page.run('document.documentElement.scrollWidth<=innerWidth'),true);
 console.log('PASS: 24-hour/ISO dates across locales, midnight, DST and offsets; compact phone layout, configurable symbols and scheduler controls (synthetic API)');
 console.log('PASS: independent group feedback, stable text through requests/queue/polls, accurate partial results');
 console.log('PASS: switch/group On/Off controls, neutral unknown state, schedule labels and singleton scheduler');
 }finally{await page.close();}
})().catch(e=>{console.error(e);process.exitCode=1;});
