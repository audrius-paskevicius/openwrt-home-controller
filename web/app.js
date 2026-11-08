'use strict';
let csrf='',signedIn=false,refreshing=false,signature='',revision=0;
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
function showLogin(){revision++;signedIn=false;csrf='';signature='';$('login').hidden=false;$('controls').hidden=true;$('logout').hidden=true;}
function statusText(state){if(!state||!state.status)return 'Position unknown';if(state.status==='Sent (position unknown)')return labels[state.action]+' sent · '+new Date(state.at*1000).toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'});if(state.status==='Queued')return labels[state.action]+' queued';return state.status;}
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
 if(state.running&&state.next){const at=new Date(state.next.at).toLocaleString([],{weekday:'short',hour:'2-digit',minute:'2-digit',timeZone:state.timezone});detail='Next: '+state.next.target+' '+state.next.action+' · '+at;}
 put(next,detail);el.querySelector('.start').disabled=state.running;el.querySelector('.stop').disabled=!state.running;
}
async function refresh(){
 if(refreshing)return;refreshing=true;const before=revision;
 try{const data=await api('state');if(before!==revision)return;csrf=data.csrf;const groups=data.groups||[],key=JSON.stringify([data.devices.map(d=>[d.id,d.name,d.icon]),groups,data.scheduler_icon]);
  if(key!==signature){$('controls').replaceChildren(...data.devices.map(d=>card(d,false)),schedulerCard(data.scheduler_icon),...groups.map(g=>card(g,true)));signature=key;}
  signedIn=true;$('login').hidden=true;$('controls').hidden=false;$('logout').hidden=false;put($('message'),'');
  for(const d of data.devices){const el=[...$('controls').children].find(x=>x.dataset.id===d.id);if(!el)continue;put(el.querySelector('.status'),statusText(d.state));el.classList.toggle('error',!!d.state.status&&!['Queued','Sent (position unknown)','Schedule cancelled'].includes(d.state.status));}
  for(const g of groups){const el=[...$('controls').children].find(x=>x.dataset.id===g.id),states=data.devices.filter(d=>g.devices.includes(d.id)).map(d=>d.state);let text='Positions unknown';const failed=states.find(s=>s.status&&!['Queued','Sent (position unknown)','Schedule cancelled'].includes(s.status));if(failed)text='Some commands failed';else if(states.some(s=>s.status==='Queued'))text='Commands queued';else if(states.length&&states.every(s=>s.status==='Sent (position unknown)'&&s.action===states[0].action))text=labels[states[0].action]+' sent to all';else if(states.some(s=>s.status))text='See individual results';put(el.querySelector('.status'),text);el.classList.toggle('error',!!failed);}
  renderScheduler(data.scheduler);
 }catch(e){if(!signedIn)showLogin();else{const el=$('scheduler');if(el){put(el.querySelector('.status'),'Connection unavailable');el.querySelectorAll('button').forEach(b=>b.disabled=true);}}if(e.message!=='login required')put($('message'),'Connection interrupted. '+e.message);}
 finally{refreshing=false;}
}
$('login').onsubmit=async event=>{event.preventDefault();const button=event.target.querySelector('button');button.disabled=true;put($('login-message'),'Signing in…');try{const data=await api('login',{user:$('username').value,password:$('password').value,remember:$('remember').checked});csrf=data.csrf;$('password').value='';put($('login-message'),'');await refresh();}catch(e){put($('login-message'),e.message);}finally{button.disabled=false;}};
$('logout').onclick=async()=>{try{await api('logout',{});showLogin();put($('message'),'');}catch(e){put($('message'),e.message);}};
document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh();});setInterval(()=>{if(signedIn&&!document.hidden)refresh();},5000);refresh();
