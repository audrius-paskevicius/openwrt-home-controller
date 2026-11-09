'use strict';
let csrf='',signedIn=false,refreshing=false,signature='',revision=0;
let disclaimer='',acknowledged=false,warningOpen=false;
const warningKey='home-controller-disclaimer';
const $=id=>document.getElementById(id),labels={up:'Up',stop:'Stop',down:'Down'};
const put=(el,text)=>{if(el.textContent!==text)el.textContent=text;};
async function api(path,body){
 const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),15000);
 try{const options={credentials:'same-origin',cache:'no-store',signal:controller.signal};
  if(body!==undefined)Object.assign(options,{method:'POST',headers:{'Content-Type':'application/json','Authorization':'Bearer '+csrf},body:JSON.stringify(body)});
  const response=await fetch('/api/'+path,options),data=await response.json();
  if(!response.ok){if(response.status===401)showLogin();throw Error(data.error||'Request failed');}return data;
 }finally{clearTimeout(timeout);}
}
function showLogin(){revision++;signedIn=false;csrf='';signature='';$('login').hidden=warningOpen;$('controls').hidden=true;$('logout').hidden=true;}
function showWarning(){
 revision++;warningOpen=true;$('login').hidden=true;$('controls').hidden=true;$('logout').hidden=true;document.querySelector('footer').hidden=true;
 const panel=$('warning');panel.replaceChildren();
 disclaimer.trim().split(/\n\s*\n/).forEach((text,i)=>{const p=document.createElement(i?'p':'h1');if(!i)p.id='warning-title';p.textContent=text;panel.append(p);});
 if(!acknowledged){const note=document.createElement('p');note.className='warning-note';note.textContent='Acknowledgement is remembered in this browser for this router while its site data is retained, including after a computer restart. It does not change the running service or scheduler.';panel.append(note);}
 const button=document.createElement('button');button.type='button';button.textContent=acknowledged?'Back':'I understand — continue';
 button.onclick=()=>{acknowledged=true;try{localStorage.setItem(warningKey,disclaimer);}catch(e){}warningOpen=false;panel.hidden=true;document.querySelector('footer').hidden=false;$('login').hidden=signedIn;$('controls').hidden=!signedIn;$('logout').hidden=!signedIn;refresh();};
 panel.append(button);panel.hidden=false;panel.focus();
}
async function initialize(){
 const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),10000);
 try{
  const response=await fetch('/disclaimer.txt',{cache:'no-store',credentials:'same-origin',signal:controller.signal});
  if(!response.ok)throw Error('HTTP '+response.status);
  disclaimer=await response.text();if(!disclaimer.trim())throw Error('The warning is empty.');
  try{acknowledged=localStorage.getItem(warningKey)===disclaimer;}catch(e){}
  $('warning').hidden=true;put($('message'),'');warningOpen=false;
  if(acknowledged){document.querySelector('footer').hidden=false;await refresh();}else showWarning();
 }catch(e){
  warningOpen=true;$('login').hidden=true;$('controls').hidden=true;$('logout').hidden=true;
  const panel=$('warning'),title=document.createElement('h1'),message=document.createElement('p'),retry=document.createElement('button');
  title.id='warning-title';title.textContent='Unable to load warning';message.textContent='Please retry before continuing. '+e.message;retry.type='button';retry.textContent='Retry';retry.onclick=()=>{retry.disabled=true;initialize();};
  panel.replaceChildren(title,message,retry);panel.hidden=false;
 }finally{clearTimeout(timeout);}
}
function dateText(value,timeZone,withDate=true){
 const date=new Date(value);if(!Number.isFinite(date.getTime()))return 'Unknown time';
 const parts=new Intl.DateTimeFormat('en-GB',{calendar:'iso8601',numberingSystem:'latn',hourCycle:'h23',
  year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',...(timeZone?{timeZone}:{})}).formatToParts(date);
 const p=Object.fromEntries(parts.map(x=>[x.type,x.value])),clock=p.hour+':'+p.minute;
 if(!withDate)return clock;
 return p.year+'-'+p.month+'-'+p.day+' '+clock;
}
function statusText(state){if(!state||!state.status)return 'Position unknown';if(state.status==='Sent (position unknown)')return labels[state.action]+' sent · '+dateText(state.at*1000,undefined,false);if(state.status==='Queued')return labels[state.action]+' queued';return state.status;}
const symbolPaths={
 blind:'M4 3h16v14H4z M4 6h16M4 9h16M4 12h16M4 15h16M20 3v18M18 21h4',
 clock:'M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20 M12 6v6l4 2',
 lamp:'M8 20h8M9 23h6M8 16c0-3-3-4-3-8a7 7 0 0 1 14 0c0 4-3 5-3 8z',
 plug:'M8 2v6M16 2v6M6 8h12v4a6 6 0 0 1-12 0zM12 18v4',
 fan:'M12 10c-5-1-6-8-1-8 4 0 5 5 2 8M14 12c3-4 10-2 7 3-2 4-7 2-8-1M11 14c2 5-4 9-6 5-2-4 2-7 5-7M12 10a2 2 0 1 0 0 4 2 2 0 0 0 0-4'
};
function heading(name,symbol,fallback){
 const row=document.createElement('div');row.className='card-heading';const title=document.createElement('h2');title.textContent=name;row.append(title);
 const kind=symbol||fallback;if(symbolPaths[kind]){const ns='http://www.w3.org/2000/svg',svg=document.createElementNS(ns,'svg'),path=document.createElementNS(ns,'path');svg.setAttribute('viewBox','0 0 24 24');svg.setAttribute('aria-hidden','true');svg.setAttribute('focusable','false');svg.classList.add('symbol');svg.dataset.symbol=kind;path.setAttribute('d',symbolPaths[kind]);svg.append(path);row.append(svg);}return row;
}
function card(item,group){
 const el=document.createElement('article');el.className='card'+(group?' group':'');el.dataset.id=item.id;
 const title=heading(item.name,item.icon,'blind');
 const status=document.createElement('p');status.className='status';status.setAttribute('aria-live','polite');status.textContent='Position unknown';
 const buttons=document.createElement('div');buttons.className='buttons';
 for(const action of ['up','stop','down']){const button=document.createElement('button');button.type='button';button.textContent=labels[action];button.className=action;button.setAttribute('aria-label',item.name+': '+labels[action]);
  button.onclick=async()=>{revision++;el.classList.remove('error');put(status,'Sending '+labels[action].toLowerCase()+'…');try{await api('command',{target:item.id,action});put(status,labels[action]+' queued');put($('message'),'');setTimeout(refresh,700);}catch(e){el.classList.add('error');put(status,e.message);}};buttons.append(button);}
 el.append(title,status,buttons);return el;
}
function schedulerCard(symbol){
 const el=document.createElement('article');el.className='card scheduler';el.id='scheduler';
 const title=heading('Scheduler',symbol,'clock');
 const status=document.createElement('p');status.className='status';status.setAttribute('aria-live','polite');
 const next=document.createElement('p');next.className='next';
 const event=document.createElement('span'),at=document.createElement('time');
 event.className='next-event';next.append(event,at);
 const buttons=document.createElement('div');buttons.className='buttons';
 for(const action of ['start','stop']){const button=document.createElement('button');button.type='button';button.className=action;button.textContent=action==='start'?'Start':'Stop';button.setAttribute('aria-label','Scheduler: '+button.textContent);
  button.onclick=async()=>{revision++;el.dataset.busy='true';buttons.querySelectorAll('button').forEach(b=>b.disabled=true);el.classList.remove('error');put(status,action==='start'?'Starting…':'Stopping…');try{const state=await api('scheduler',{action});delete el.dataset.busy;renderScheduler(state);}catch(e){delete el.dataset.busy;el.classList.add('error');put(status,e.message);buttons.querySelectorAll('button').forEach(b=>b.disabled=false);}};buttons.append(button);}
 el.append(title,status,next,buttons);return el;
}
function renderScheduler(state){
 const el=$('scheduler');if(!el||el.dataset.busy)return;
 const status=el.querySelector('.status'),next=el.querySelector('.next');el.classList.remove('error');put(status,state.running?'Running':'Stopped');status.classList.toggle('running',state.running);
 let detail=state.running?'No schedules enabled':'Manual controls remain available';
 if(state.running&&state.enabled_schedules)detail='No upcoming event in the next week';
 const at=next.querySelector('time'),hasNext=!!(state.running&&state.next);
 at.hidden=!hasNext;
 if(hasNext){detail='Next: '+state.next.target+' '+(labels[state.next.action]||state.next.action);put(at,dateText(state.next.at,state.timezone));at.dateTime=state.next.at;at.title=(state.timezone||Intl.DateTimeFormat().resolvedOptions().timeZone)+' · '+state.next.at;}
 put(next.querySelector('.next-event'),detail);el.querySelector('.start').disabled=state.running;el.querySelector('.stop').disabled=!state.running;
}
async function refresh(){
 if(refreshing||!acknowledged||warningOpen)return;refreshing=true;const before=revision;
 try{const data=await api('state');if(before!==revision)return;csrf=data.csrf;const groups=data.groups||[],key=JSON.stringify([data.devices.map(d=>[d.id,d.name,d.icon]),groups,data.scheduler_icon,data.control_order]);
  if(key!==signature){const cards=new Map([...data.devices.map(d=>['device:'+d.id,card(d,false)]),['scheduler',schedulerCard(data.scheduler_icon)],...groups.map(g=>['group:'+g.id,card(g,true)])]);const order=[...new Set([...(data.control_order||[]),...cards.keys()])];$('controls').replaceChildren(...order.filter(id=>cards.has(id)).map(id=>cards.get(id)));signature=key;}
  signedIn=true;$('login').hidden=true;$('controls').hidden=false;$('logout').hidden=false;put($('message'),'');
  for(const d of data.devices){const el=[...$('controls').children].find(x=>x.dataset.id===d.id);if(!el)continue;put(el.querySelector('.status'),statusText(d.state));el.classList.toggle('error',!!d.state.status&&!['Queued','Sent (position unknown)','Schedule cancelled'].includes(d.state.status));}
  for(const g of groups){const el=[...$('controls').children].find(x=>x.dataset.id===g.id),states=data.devices.filter(d=>g.devices.includes(d.id)).map(d=>d.state);let text='Positions unknown';const failed=states.find(s=>s.status&&!['Queued','Sent (position unknown)','Schedule cancelled'].includes(s.status));if(failed)text='Some commands failed';else if(states.some(s=>s.status==='Queued'))text='Commands queued';else if(states.length&&states.every(s=>s.status==='Sent (position unknown)'&&s.action===states[0].action))text=labels[states[0].action]+' sent to all';else if(states.some(s=>s.status))text='See individual results';put(el.querySelector('.status'),text);el.classList.toggle('error',!!failed);}
  renderScheduler(data.scheduler);
 }catch(e){if(!signedIn)showLogin();else{const el=$('scheduler');if(el){put(el.querySelector('.status'),'Connection unavailable');el.querySelectorAll('button').forEach(b=>b.disabled=true);}}if(e.message!=='login required')put($('message'),'Connection interrupted. '+e.message);}
 finally{refreshing=false;}
}
$('login').onsubmit=async event=>{event.preventDefault();const button=event.target.querySelector('button');button.disabled=true;put($('login-message'),'Signing in…');try{const data=await api('login',{user:$('username').value,password:$('password').value,remember:$('remember').checked});csrf=data.csrf;$('password').value='';put($('login-message'),'');await refresh();}catch(e){put($('login-message'),e.message);}finally{button.disabled=false;}};
$('logout').onclick=async()=>{try{await api('logout',{});showLogin();put($('message'),'');}catch(e){put($('message'),e.message);}};
$('warning-link').onclick=event=>{if(disclaimer){event.preventDefault();showWarning();}};
document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh();});setInterval(()=>{if(signedIn&&!document.hidden)refresh();},5000);initialize();
