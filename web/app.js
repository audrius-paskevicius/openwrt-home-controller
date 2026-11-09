'use strict';
let csrf='',signedIn=false,refreshing=false,signature='',revision=0;
let disclaimer='',acknowledged=false,warningOpen=false;
let lastRefreshAt=0,fastRefreshUntil=0;
const warningKey='home-controller-disclaimer';
const $=id=>document.getElementById(id),labels={up:'Up',stop:'Stop',down:'Down',on:'On',off:'Off',toggle:'Toggle',dim:'Dim',brightness:'Brightness',temperature:'White temperature',color:'Colour',hue:'Colour hue',saturation:'Saturation'};
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
function statusText(state,type){if(!state||!state.status)return type==='switch'||type==='dimmer'?'State unknown':'Position unknown';if(state.status==='Sent (position unknown)')return labels[state.action]+' sent · '+dateText(state.at*1000,undefined,false);if(state.status==='Queued')return labels[state.action]+' queued';return state.status;}
function groupStatusText(state){
 if(!state?.status)return '';
 const action=labels[state.action]||state.action,time=' · '+dateText(state.at*1000,undefined,false);
 if(state.sent===state.total)return action+' sent to all'+time;
 if(state.sent)return action+' sent to '+state.sent+'/'+state.total+time;
 return action+(state.failed?' failed':' cancelled')+time;
}
function renderResult(el,state,group){
 if(!el||el.dataset.busy)return;
 // Keep the last result during short queue/transmit stages. Only changed,
 // completed results touch the live region, including after a request error.
 if(state?.status==='Queued')return;
 const key=JSON.stringify(state||{});if(el.dataset.result===key)return;
 el.dataset.result=key;
 put(el.querySelector('.status'),group?groupStatusText(state):statusText(state,el.dataset.type));
 el.classList.toggle('error',group?!!state?.failed:!!state?.status&&!['Sent (position unknown)','Schedule cancelled','Superseded'].includes(state.status));
}
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
 if(item.type==='ikea')return ikeaCard(item);
 const el=document.createElement('article');el.className='card'+(group?' group':'');el.dataset.id=item.id;el.dataset.type=item.type||'blind';
 const title=heading(item.name,item.icon,item.type==='dimmer'?'lamp':item.type==='switch'?'plug':'blind');
 const status=document.createElement('p');status.className='status';status.setAttribute('aria-live','polite');status.textContent=group?'':['switch','dimmer'].includes(item.type)?'State unknown':'Position unknown';
 const buttons=document.createElement('div');buttons.className='buttons';
 for(const action of item.type==='dimmer'?['toggle','dim']:item.type==='switch'?['on','off']:['up','stop','down']){const button=document.createElement('button');button.type='button';button.textContent=labels[action];button.className=action;button.setAttribute('aria-label',item.name+': '+labels[action]);if(action==='dim')button.title='Start or stop changing brightness while the lamp is on';if(action==='toggle')button.title='Toggle power on or off';
  button.onclick=async()=>{if(item.type==='dimmer'&&el.dataset.busy)return;revision++;const request=el.request=(el.request||0)+1;el.dataset.busy='true';el.setAttribute('aria-busy','true');try{await api('command',{target:item.id,action});if(request===el.request){put($('message'),'');setTimeout(refresh,700);}}catch(e){if(request===el.request){el.classList.add('error');put(status,e.message);}}finally{if(request===el.request){delete el.dataset.busy;el.removeAttribute('aria-busy');}}};buttons.append(button);}
 el.append(title,status,buttons);return el;
}
function ikeaCard(item){
 const info=item.ikea||{},caps=info.capabilities||{},hasSliders=!!caps.brightness;
 const el=document.createElement('article');el.className='card ikea'+(info.kind==='group'?' group':'');el.dataset.id=item.id;el.dataset.type='ikea';
 const body=document.createElement('div');body.className='ikea-body';const main=document.createElement('div');main.className='ikea-main';
 const status=document.createElement('p');status.className='status';status.setAttribute('aria-live','polite');
 const power=document.createElement('div');power.className='buttons';
 const send=async(action,extra={})=>{revision++;fastRefreshUntil=Date.now()+10000;const request=el.request=(el.request||0)+1;el.dataset.busy='true';delete el.dataset.clientError;try{await api('command',{target:item.id,action,...extra});fastRefreshUntil=Date.now()+10000;setTimeout(refresh,250);}catch(e){if(request===el.request){el.dataset.clientError=e.message;el.querySelectorAll('input').forEach(x=>{delete x.pending;});if(el.dataset.available==='true'){put(status,e.message);el.classList.add('error');}}}finally{if(request===el.request)delete el.dataset.busy;}};
 for(const action of ['on','off']){const b=document.createElement('button');b.type='button';b.className=action;b.textContent=labels[action];b.setAttribute('aria-label',item.name+': '+labels[action]);b.onclick=()=>send(action);power.append(b);}
 main.append(heading(item.name,item.icon,hasSliders?'lamp':'plug'),status,power);body.append(main);
 const sliders=document.createElement('div');sliders.className='ikea-sliders';
 for(const [action,letter,min,max,step,initial] of [['brightness','B',0,100,1,100],['temperature','W',2200,4000,50,2700]]){
  if(!caps[action])continue;
  const slider=document.createElement('label');slider.className='ikea-slider';
  const rail=document.createElement('span');rail.className='slider-rail';rail.setAttribute('aria-hidden','true');
  const knob=document.createElement('span');knob.className='slider-knob';knob.textContent=letter;rail.append(knob);
  const input=document.createElement('input');input.type='range';input.dataset.action=action;input.min=min;input.max=max;input.step=step;input.value=initial;
  input.setAttribute('aria-label',item.name+': '+labels[action]);input.setAttribute('aria-orientation','horizontal');
  input.onpointerdown=input.onkeydown=()=>{input.editing=true;};
  input.onpointerup=input.onpointercancel=input.onkeyup=input.onblur=()=>{input.editing=false;};
  input.oninput=()=>paintIKEASlider(input);input.onchange=()=>{paintIKEASlider(input);input.pending={value:Number(input.value),until:Date.now()+8000};send(action,{value:Number(input.value)});};
  slider.append(rail,input);sliders.append(slider);paintIKEASlider(input);
 }
 if(hasSliders)body.append(sliders);
 if(caps.color){
  const colors=document.createElement('div');colors.className='ikea-colors';colors.setAttribute('role','group');colors.setAttribute('aria-label',item.name+': Colour');
  for(const [name,color] of [['Red','#ff0000'],['Yellow','#ffff00'],['Green','#00ff00'],['Blue','#0000ff'],['Purple','#8000ff']]){
   const b=document.createElement('button');b.type='button';b.className='color-dot '+name.toLowerCase();b.dataset.color=color;b.title=name;b.setAttribute('aria-label',item.name+': '+name);b.onclick=()=>send('color',{color});colors.append(b);
  }
  body.append(colors);
 }
 el.append(body);renderIKEA(el,item);return el;
}
function paintIKEASlider(input,unknown=false){
 const action=input.dataset.action,value=Number(input.value),text=unknown?'Mixed':value+(action==='temperature'?'K':action==='hue'?'°':'%');
 input.parentNode.style.setProperty('--position',(value-Number(input.min))/(Number(input.max)-Number(input.min)));
 input.setAttribute('aria-valuetext',text);input.title=labels[action]+': '+text;
}
function ikeaStatusText(live){
 if(!live.reachable)return 'Unavailable';if(live.partial)return 'Status incomplete';if(live.power===false)return 'Off';if(live.power!==true)return 'Mixed';
 let text='On';if(live.brightness>=0&&live.brightness!=null)text+=' '+live.brightness+'%';else if(live.brightness===-1)text+=' · Mixed';
 if(live.mode==='white'&&live.temperature>0)text+=' '+live.temperature+'K';
 return text;
}
function renderIKEA(el,item){
 if(!el)return;const live=item.ikea||{},available=!!live.reachable;el.dataset.available=String(available);
 let text=ikeaStatusText(live);
 // One bulb can report before another. Confirm a new mixed group state on a
 // later poll; persistent differences and unavailable members remain visible.
 if(live.kind==='group'&&live.reachable&&!live.partial&&(live.power==null||live.brightness===-1)&&el.stableStatus){
  el.mixedSince??=Date.now();if(Date.now()-el.mixedSince<4000)text=el.stableStatus;
 }else{delete el.mixedSince;el.stableStatus=text;}
 const failure=item.state?.status&&!['Queued','Sent (position unknown)','Superseded','Schedule cancelled'].includes(item.state.status)?item.state.status:'';
 // Current unavailability takes precedence over an earlier command outcome.
 const error=available?(el.dataset.clientError||failure):'';
 if(error)text=error;
 put(el.querySelector('.status'),text);el.classList.toggle('error',!!error);
 el.querySelectorAll('button,input').forEach(input=>{input.disabled=!available;});
 for(const action of ['brightness','temperature']){
  const input=el.querySelector('[data-action="'+action+'"]');if(!input||el.dataset.busy||input.editing)continue;
  const value=live.controls?.[action]??live[action],known=available&&value!=null&&value!==-1;
  if(input.pending){
   if(!available||failure||el.dataset.clientError||Date.now()>=input.pending.until||(known&&value===input.pending.value))delete input.pending;
   else continue;
  }
  if(known)input.value=String(value);paintIKEASlider(input,!known);
 }
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
 if(hasNext){detail='Next: '+state.next.target+' '+(labels[state.next.action]||state.next.action)+(state.next.value!=null?' '+state.next.value+(state.next.action==='temperature'?' K':state.next.action==='hue'?'°':'%'):state.next.color?' '+state.next.color.toUpperCase():'');put(at,dateText(state.next.at,state.timezone));at.dateTime=state.next.at;at.title=(state.timezone||Intl.DateTimeFormat().resolvedOptions().timeZone)+' · '+state.next.at;}
 put(next.querySelector('.next-event'),detail);el.querySelector('.start').disabled=state.running;el.querySelector('.stop').disabled=!state.running;
}
async function refresh(){
 if(refreshing||!acknowledged||warningOpen)return;refreshing=true;lastRefreshAt=Date.now();const before=revision;
 try{const data=await api('state');if(before!==revision)return;csrf=data.csrf;const groups=data.groups||[],key=JSON.stringify([data.devices.map(d=>[d.id,d.name,d.icon,d.type,d.ikea?.capabilities]),groups,data.scheduler_icon,data.control_order]);
  if(key!==signature){const cards=new Map([...data.devices.map(d=>['device:'+d.id,card(d,false)]),['scheduler',schedulerCard(data.scheduler_icon)],...groups.map(g=>['group:'+g.id,card(g,true)])]);const order=[...new Set([...(data.control_order||[]),...cards.keys()])];$('controls').replaceChildren(...order.filter(id=>cards.has(id)).map(id=>cards.get(id)));signature=key;}
  signedIn=true;$('login').hidden=true;$('controls').hidden=false;$('logout').hidden=false;put($('message'),'');
  for(const d of data.devices){const el=[...$('controls').children].find(x=>x.dataset.id===d.id);if(d.type==='ikea')renderIKEA(el,d);else renderResult(el,d.state,false);}
  for(const g of groups)renderResult([...$('controls').children].find(x=>x.dataset.id===g.id),data.group_states?.[g.id],true);
  renderScheduler(data.scheduler);
 }catch(e){if(!signedIn)showLogin();else{const el=$('scheduler');if(el){put(el.querySelector('.status'),'Connection unavailable');el.querySelectorAll('button').forEach(b=>b.disabled=true);}}if(e.message!=='login required')put($('message'),'Connection interrupted. '+e.message);}
 finally{refreshing=false;}
}
$('login').onsubmit=async event=>{event.preventDefault();const button=event.target.querySelector('button');button.disabled=true;put($('login-message'),'Signing in…');try{const data=await api('login',{user:$('username').value,password:$('password').value,remember:$('remember').checked});csrf=data.csrf;$('password').value='';put($('login-message'),'');await refresh();}catch(e){put($('login-message'),e.message);}finally{button.disabled=false;}};
$('logout').onclick=async()=>{try{await api('logout',{});showLogin();put($('message'),'');}catch(e){put($('message'),e.message);}};
$('warning-link').onclick=event=>{if(disclaimer){event.preventDefault();showWarning();}};
document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh();});setInterval(()=>{if(signedIn&&!document.hidden&&Date.now()-lastRefreshAt>=(Date.now()<fastRefreshUntil?900:4900))refresh();},1000);initialize();
