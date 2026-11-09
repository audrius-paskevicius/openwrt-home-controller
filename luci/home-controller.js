'use strict';
'require view';
'require rpc';
'require ui';
'require form';
'require poll';
'require home-controller-disclaimer as warning';

const get=rpc.declare({object:'home-controller',method:'get',expect:{}});
const apply=rpc.declare({object:'home-controller',method:'save',params:['config','controller_instance'],expect:{}});
const stage=rpc.declare({object:'home-controller',method:'stage',params:['config','controller_instance'],expect:{}});
const discard=rpc.declare({object:'home-controller',method:'discard',expect:{}});
const password=rpc.declare({object:'home-controller',method:'password',params:['user','password'],expect:{}});
const revoke=rpc.declare({object:'home-controller',method:'revoke',expect:{}});
const sessions=rpc.declare({object:'home-controller',method:'sessions',expect:{}});
const tls=rpc.declare({object:'home-controller',method:'tls',expect:{}});
const clone=x=>JSON.parse(JSON.stringify(x));
const symbols={none:'None',blind:'Blind',clock:'Clock',lamp:'Lamp',plug:'Plug',fan:'Fan'};
const kinds={device:'Device',group:'Group',scheduler:'Scheduler'};
const actions={up:'Up',stop:'Stop',down:'Down'};
const events={time:'Fixed time',sunrise:'Sunrise',sunset:'Sunset',dawn:'Civil dawn',dusk:'Civil dusk'};
const weekdays={1:'Mon',2:'Tue',3:'Wed',4:'Thu',5:'Fri',6:'Sat',0:'Sun'};
function result(r){if(r.error)throw Error(r.error);return r;}
function feedback(){return E('p',{role:'status','aria-live':'polite',hidden:''});}
function tell(node,text,error){node.textContent=text;node.hidden=!text;node.classList.toggle('error',!!error);node.classList.toggle('alert-message',!!error);}
function key(item){return item.kind==='scheduler'?'scheduler':item.kind+':'+item.id;}
function list(value){return Array.isArray(value)?value:value?[value]:[];}
function choice(section,type,name,title,values,description){
 const o=section.option(type,name,title,description);Object.entries(values).forEach(([k,v])=>o.value(k,v));o.rmempty=false;
 o.textvalue=function(sid){return list(this.cfgvalue(sid)).map(v=>values[v]||v).join(', ');};return o;
}
function value(section,name,title,datatype,description){const o=section.option(form.Value,name,title,description);o.rmempty=false;if(datatype)o.datatype=datatype;return o;}
function dummy(section,name,title,render){const o=section.option(form.DummyValue,name,title);o.renderWidget=()=>E('div',{},render());return o;}
function webServerHelp(description){
 const text='Home Controller uses a separate uHTTPd instance, started and stopped with this service. This port serves the control page only. LuCI uses the router’s main uHTTPd instance, configured separately under Services → uHTTPd.';
 let pinned=false,dismissed=false;
 const target=E('span',{class:'hc-setting-help',role:'button',tabindex:'0','data-tooltip':text,
  'data-tooltip-style':'hc-help-tooltip','aria-label':text},description.textContent);
 const show=()=>{
  dismissed=false;
  ui.showTooltip({target});
  const tip=document.querySelector('body > .cbi-tooltip');
  if(tip){const r=tip.getBoundingClientRect();tip.style.left=window.scrollX+Math.max(8,Math.min(r.left,window.innerWidth-r.width-8))+'px';}
 };
 const hide=()=>{pinned=false;dismissed=true;ui.hideTooltip({target});};
 const toggle=()=>{const wasPinned=pinned;target.focus();if(wasPinned)hide();else{pinned=true;show();}};
 description.replaceChildren(target);
 description.addEventListener('mouseover',show);
 description.addEventListener('mouseout',()=>{if(!dismissed&&document.activeElement===target)show();});
 description.addEventListener('click',ev=>{ev.preventDefault();toggle();});
 target.addEventListener('focus',show);target.addEventListener('blur',hide);
 target.addEventListener('keydown',ev=>{
  if(ev.key==='Escape'){ev.preventDefault();ev.stopPropagation();hide();}
  else if(ev.key==='Enter'||ev.key===' '){ev.preventDefault();toggle();}
 });
}
function command(section,name,title,caption,handler){
 const message=feedback(),o=section.option(form.Button,name,title);let busy=false;o.inputtitle=caption;o.inputstyle='action';
 o.renderWidget=function(...args){return E('div',{},[form.Button.prototype.renderWidget.apply(this,args),message]);};
 o.onclick=async function(ev){
  const button=ev.currentTarget;if(busy)return;busy=true;button.disabled=true;tell(message,'Working…');
  try{tell(message,await handler()||'');}catch(e){tell(message,e.message||String(e),true);}finally{busy=false;button.disabled=false;}
 };return o;
}
function model(c){
 const controls=[...c.devices.map(d=>({...d,kind:'device',icon:d.icon||'blind'})),
  {id:'scheduler',name:'Scheduler',kind:'scheduler',icon:c.scheduler_icon||'clock'},...c.groups.map(g=>({...g,kind:'group',icon:g.icon||'blind'}))];
 const order=[...new Set([...(c.control_order||[]),...controls.map(key)])];controls.sort((a,b)=>order.indexOf(key(a))-order.indexOf(key(b)));
 return {general:{enabled:c.enabled?'1':'0'},control:controls.map((d,i)=>({...d,'.name':'control_'+i})),
  schedule:c.schedules.map((s,i)=>({...s,enabled:s.enabled?'1':'0',days:s.days.map(String),'.name':'schedule_'+i})),
  scheduling:{latitude:c.latitude,longitude:c.longitude,timezone:c.timezone},access:{port:c.port,networks:c.networks,session_days:c.session_days},advanced:{preferred_controller:c.preferred_controller||'automatic'}};
}
function configuration(m,original){
 const c=clone(original),controls=m.data.sections('json','control'),settings=m.data.get('json','scheduling');
 c.enabled=m.data.get('json','general','enabled')==='1';c.latitude=Number(settings.latitude);c.longitude=Number(settings.longitude);c.timezone=settings.timezone;
 c.port=Number(m.data.get('json','access','port'));c.networks=list(m.data.get('json','access','networks'));c.session_days=Number(m.data.get('json','access','session_days'));
 c.preferred_controller=m.data.get('json','advanced','preferred_controller')||'automatic';if(c.preferred_controller==='automatic')c.preferred_controller='';
 c.devices=controls.filter(d=>d.kind==='device').map(d=>({id:d.id,name:d.name.trim(),icon:d.icon,radio_id:Number(d.radio_id),house:Number(d.house),unit:Number(d.unit)}));
 c.groups=controls.filter(d=>d.kind==='group').map(d=>({id:d.id,name:d.name.trim(),icon:d.icon,devices:list(d.devices)}));
 c.scheduler_icon=controls.find(d=>d.kind==='scheduler').icon;c.control_order=controls.map(key);
 c.schedules=m.data.sections('json','schedule').map(s=>({id:s.id,enabled:s.enabled==='1',target:s.target,action:s.action,days:list(s.days).map(Number),event:s.event,
  time:s.event==='time'?s.time:'',offset:s.event==='time'?0:Number(s.offset||0),earliest:s.event==='time'?'':s.earliest||'',latest:s.event==='time'?'':s.latest||'',fallback:s.event==='time'?'':s.fallback||''}));
 return c;
}
function dateText(value,timeZone,withDate=true){
 const date=new Date(value);if(!Number.isFinite(date.getTime()))return 'Unknown time';
 const parts=new Intl.DateTimeFormat('en-GB',{calendar:'iso8601',numberingSystem:'latn',hourCycle:'h23',
  year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',...(timeZone?{timeZone}:{})}).formatToParts(date);
 const p=Object.fromEntries(parts.map(x=>[x.type,x.value])),clock=p.hour+':'+p.minute;
 if(!withDate)return clock;
 return p.year+'-'+p.month+'-'+p.day+' '+clock;
}
function nextText(state){
 if(!state)return 'Waiting for service status';if(!state.running)return 'Stopped';if(!state.enabled_schedules)return 'Running · no schedules enabled';if(!state.next)return 'Running · no event in the next week';
 return 'Running · '+state.next.target+' · '+actions[state.next.action]+' · '+dateText(state.next.at,state.timezone);
}
return view.extend({
 load(){return Promise.all([get().then(result),warning.load()]).then(([data,text])=>({...data,disclaimer:text}));},
 render(data){this.disclaimer=data.disclaimer;return warning.gate(data.disclaimer,()=>this.renderSettings(data));},
 async renderSettings(data){
  this.original=clone(data.config);const m=this.map=new form.JSONMap(model(data.config),'Home Controller');m.tabbed=true;let fresh=data,preferred=null,choiceSignature='';
  const s=m.section(form.NamedSection,'general','general','General');s.hidetitle=true;
  const enabled=s.option(form.Flag,'enabled','Enable service','To pause scheduled actions only, use Scheduler on the control page.');enabled.rmempty=false;
  const service=E('p',{role:'status',hidden:'',class:'alert-message warning'}),controller=E('div'),scheduler=E('span'),links=E('span');
  dummy(s,'_controller','TellSticks',()=>controller);dummy(s,'_scheduler','Scheduler',()=>scheduler);dummy(s,'_links','Control page',()=>links);
  const updateStatus=d=>{
   const previousPreference=fresh.config.preferred_controller||'automatic';
   fresh=d;this.connectedControllers=d.controllers||[];const active=d.active_config||d.config;service.hidden=true;service.textContent='';scheduler.textContent=active.enabled?nextText(d.status?.scheduler):'Service is disabled';
   const schedule=d.status?.scheduler;scheduler.title=active.enabled&&schedule?.running&&schedule.next?(schedule.timezone||Intl.DateTimeFormat().resolvedOptions().timeZone)+' · '+schedule.next.at:'';
   const devices=d.controllers||[],titles=['Model','Serial','Capabilities','Use'];
   controller.replaceChildren(devices.length?E('table',{class:'table'},[
    E('tr',{class:'tr table-titles'},titles.map(t=>E('th',{class:'th'},t))),
    ...devices.map(usb=>E('tr',{class:'tr'},[usb.product||'TellStick',usb.serial||'Unavailable',usb.capability||'Unknown',d.active_controller?.serial===usb.serial?'In use':'Available'].map((v,i)=>E('td',{class:'td','data-title':titles[i]},v))))
   ]):E('span',{},'No TellStick connected'));
   if(devices.some(usb=>usb.capability==='Transmit and receive'))controller.append(E('p',{class:'cbi-section-descr'},'Home Controller currently uses sending only.'));
   if(preferred){
    const choices={automatic:'Automatic',...Object.fromEntries(devices.filter(usb=>usb.serial).map(usb=>[usb.serial,(usb.product||'TellStick')+' ('+usb.serial+')']))};
    const saved=d.config.preferred_controller||'automatic',current=preferred.formvalue('advanced')||'automatic';
    const next=Object.hasOwn(choices,current)?(current===previousPreference?saved:current):'automatic';
    preferred.keylist=[];preferred.vallist=[];Object.entries(choices).forEach(([k,v])=>preferred.value(k,v));
    m.data.set('json','advanced','preferred_controller',Object.hasOwn(choices,saved)?saved:'automatic');
    this.original.preferred_controller=d.config.preferred_controller||'';
    const signature=JSON.stringify(choices),select=m.root?.querySelector('[id="'+preferred.cbid?.('advanced')+'"] select');
    if(select&&signature!==choiceSignature)select.replaceChildren(...Object.entries(choices).map(([v,label])=>E('option',{value:v},label)));
    preferred.getUIElement('advanced')?.setValue(Object.hasOwn(choices,next)?next:'automatic');
    choiceSignature=signature;
   }
   const address=d.access?.listeners?.[0];links.replaceChildren(address?E('a',{href:'https://'+address+'/',target:'_blank',rel:'noopener'},'Open controls'):'Unavailable while the service is stopped');
  };updateStatus(data);
  poll.add(()=>{if(!m.root?.isConnected)return;return get().then(result).then(updateStatus).catch(()=>{service.textContent='Unable to refresh status';service.hidden=false;});},3);
  const controls=m.section(form.GridSection,'control','Controls','Drag rows to arrange the control page. Devices, groups, and Scheduler can appear in any order.');
  controls.hidetitle=true;controls.anonymous=true;controls.addremove=true;controls.sortable=true;controls.nodescriptions=true;controls.addbtntitle='Add device or group';controls.modaltitle=sid=>kinds[m.data.get('json',sid,'kind')]||'New control';
  let o=value(controls,'name','Name','maxlength(80)');o.validate=(_sid,v)=>v.trim()?true:'Enter a name.';
  choice(controls,form.ListValue,'kind','Type',kinds);choice(controls,form.ListValue,'icon','Symbol',symbols);
  o=value(controls,'house','House code','and(uinteger,range(1,65535))','Hasta selflearningv2 radio address.');o.modalonly=true;o.depends('kind','device');
  o=value(controls,'unit','Unit','and(uinteger,range(0,15))');o.modalonly=true;o.depends('kind','device');
  o=controls.option(form.MultiValue,'devices','Devices');o.modalonly=true;o.rmempty=false;o.depends('kind','group');
  controls.addModalOptions=function(section,sid){
   const d=m.data.get('json',sid),type=section.getOption('kind'),members=section.getOption('devices');type.readonly=m.addedSection!==sid;type.keylist=[];type.vallist=[];
   Object.entries(kinds).filter(([k])=>k!=='scheduler'||d.kind==='scheduler').forEach(([k,v])=>type.value(k,v));section.getOption('name').readonly=d.kind==='scheduler';
   members.keylist=[];members.vallist=[];m.data.sections('json','control').filter(x=>x.kind==='device').forEach(x=>members.value(x.id,x.name));
  };
  controls.handleAdd=function(){
   const id='control_'+Date.now().toString(36)+'_'+Math.random().toString(36).slice(2,6),sid=m.data.add('json','control',id),used=new Set(m.data.sections('json','control').map(x=>Number(x.radio_id)));let radio=1;while(used.has(radio))radio++;
   Object.entries({id,kind:'device',name:'',icon:'blind',radio_id:radio,unit:1}).forEach(([k,v])=>m.data.set('json',sid,k,v));m.addedSection=sid;return this.renderMoreOptionsModal(sid);
  };
  controls.renderRowActions=function(sid){const node=form.GridSection.prototype.renderRowActions.call(this,sid);if(m.data.get('json',sid,'kind')==='scheduler')node.querySelector('.cbi-button-remove')?.remove();return node;};
  controls.handleRemove=function(sid){
   const d=m.data.get('json',sid);if(d.kind==='scheduler')return;const used=m.data.sections('json','control').filter(x=>x.kind==='group'&&list(x.devices).includes(d.id)).map(x=>x.name);
   if(m.data.sections('json','schedule').some(x=>x.target===d.id))used.push('a schedule');
   if(used.length){ui.showModal('Control is in use',[E('p',{},'Used by '+used.join(', ')+'. Edit those references first.'),E('div',{class:'right'},E('button',{class:'cbi-button',click:ui.hideModal},'Close'))]);return;}
   return form.GridSection.prototype.handleRemove.call(this,sid);
  };
  const scheduling=m.section(form.NamedSection,'scheduling','scheduling','Schedules');scheduling.hidetitle=true;
  const grid=scheduling.option(form.SectionValue,'_schedules',form.GridSection,'schedule').subsection;grid.anonymous=true;grid.addremove=true;grid.nodescriptions=true;grid.addbtntitle='Add schedule';grid.modaltitle='Schedule';
  o=grid.option(form.Flag,'enabled','Enabled');o.rmempty=false;o.editable=true;
  choice(grid,form.ListValue,'target','Device or group',{});choice(grid,form.ListValue,'action','Action',actions);
  o=choice(grid,form.ListValue,'event','When',events);o.textvalue=function(sid){const s=m.data.get('json',sid);return events[s.event]+(s.event!=='time'&&Number(s.offset)?' '+(s.offset>0?'+':'−')+Math.abs(s.offset)+' min':'');};
  o=choice(grid,form.MultiValue,'days','Days',weekdays);o.validate=(_sid,v)=>list(v).length?true:'Choose at least one weekday.';
  o=value(grid,'time','Time');o.placeholder='08:00';o.depends('event','time');o.validate=(_sid,v)=>/^([01]\d|2[0-3]):[0-5]\d$/.test(v)?true:'Use HH:MM (24-hour time).';
  o=value(grid,'offset','Offset (minutes)','and(integer,range(-720,720))','Negative values run before the solar event; positive values run after it.');o.default='0';o.modalonly=true;Object.keys(events).filter(k=>k!=='time').forEach(k=>o.depends('event',k));
  for(const [name,title] of [['earliest','Not before'],['latest','Not after'],['fallback','Fallback time']]){
   o=value(grid,name,title,null,name==='fallback'?'Used when the solar event does not occur. Leave empty to skip that day.':null);o.rmempty=true;o.modalonly=true;o.placeholder='HH:MM';o.validate=(_sid,v)=>!v||/^([01]\d|2[0-3]):[0-5]\d$/.test(v)?true:'Use HH:MM (24-hour time).';Object.keys(events).filter(k=>k!=='time').forEach(k=>o.depends('event',k));
  }
  const targets=()=>m.data.sections('json','control').filter(x=>x.kind!=='scheduler');
  for(const section of [controls,grid])section.renderMoreOptionsModal=async function(...args){await m.parse();return form.GridSection.prototype.renderMoreOptionsModal.apply(this,args);};
  grid.addModalOptions=function(section){const t=section.getOption('target');t.keylist=[];t.vallist=[];targets().forEach(d=>t.value(d.id,d.name));};
  grid.getOption('target').textvalue=function(sid){return targets().find(x=>x.id===m.data.get('json',sid,'target'))?.name||'Missing device';};
  grid.handleAdd=function(){const id='schedule_'+Date.now().toString(36)+'_'+Math.random().toString(36).slice(2,6),sid=m.data.add('json','schedule',id);Object.entries({id,enabled:'0',target:targets()[0]?.id||'',action:'up',days:['1','2','3','4','5'],event:'time',time:'08:00',offset:0}).forEach(([k,v])=>m.data.set('json',sid,k,v));m.addedSection=sid;return this.renderMoreOptionsModal(sid);};
  value(scheduling,'timezone','Time zone',null,'Use “router” to follow the router’s time zone.').value('router','Follow router');
  value(scheduling,'latitude','Latitude','range(-90,90)');value(scheduling,'longitude','Longitude','range(-180,180)');
  dummy(scheduling,'_sunwait','Solar schedules',()=>data.sunwait?'Sunwait is installed':'Install sunwait to use sunrise, sunset, dawn, and dusk.');
  const access=m.section(form.NamedSection,'access','access','Access');access.hidetitle=true;
  o=value(access,'port','HTTPS port','and(uinteger,range(1024,65535))','Separate web server for the control page.');
  o.renderFrame=function(...args){const node=form.Value.prototype.renderFrame.apply(this,args);const description=node.querySelector('.cbi-value-description');if(description)webServerHelp(description);return node;};
  o=choice(access,form.MultiValue,'networks','Allowed networks',Object.fromEntries([...new Set(['lan',...(data.networks||[]),...data.config.networks])].filter(n=>n!=='wan'&&n!=='wan6').map(n=>[n,n])),'Allow access only through the selected LAN or VPN networks. Unchecked VPNs are excluded. WAN is unavailable. Existing firewall restrictions still apply.');
  o.validate=(_sid,v)=>list(v).length?true:'Select at least one LAN or VPN network.';
  command(access,'_tls','HTTPS certificate','Renew certificate',async()=>{result(await tls());return 'Certificate renewed. Home Controller is restarting.';});
  const username=value(access,'_username','Username',null,'Shared login for the control page.');username.cfgvalue=()=>data.account||'home';username.write=username.remove=()=>{};
  const pass=value(access,'_password','New password');pass.password=true;pass.rmempty=true;pass.cfgvalue=()=>'';pass.write=pass.remove=()=>{};
  command(access,'_password_action','Shared login','Set password',async()=>{
   const user=username.formvalue('access').trim(),secret=pass.formvalue('access')||'';if(!secret)throw Error('Enter a new password.');if([...secret].length>256)throw Error('Password must be at most 256 characters.');
   if(!/^[a-z][a-z0-9_-]{0,31}$/.test(user))throw Error('Use a username starting with a lowercase letter, then lowercase letters, numbers, hyphens or underscores (up to 32 characters).');
   result(await password(user,secret));data.account=user;pass.getUIElement('access').setValue('');return 'Password changed. Previous sessions have been signed out.';
  });
  value(access,'session_days','Remember sign-in (days)','and(uinteger,range(1,90))');const sessionList=E('div');
  command(access,'_sessions','Signed-in devices','Show devices',async()=>{
   const items=result(await sessions()).sessions,rows=items.map(s=>E('tr',{class:'tr'},[s.device||'Browser',...['created','expires'].map(k=>E('time',{datetime:new Date(s[k]*1000).toISOString(),title:Intl.DateTimeFormat().resolvedOptions().timeZone+' · '+new Date(s[k]*1000).toISOString()},dateText(s[k]*1000)))].map(v=>E('td',{class:'td'},v))));
   sessionList.replaceChildren(items.length?E('table',{class:'table'},[E('tr',{class:'tr table-titles'},['Device','Signed in','Expires'].map(t=>E('th',{class:'th'},t))),...rows]):E('p',{},'No active sessions.'));return '';
  });
  dummy(access,'_session_list','',()=>sessionList);command(access,'_revoke','Sessions','Sign out all devices',async()=>{result(await revoke());sessionList.replaceChildren();return 'All sessions have been signed out.';});
  const advanced=m.section(form.NamedSection,'advanced','advanced','Advanced');advanced.hidetitle=true;
  preferred=choice(advanced,form.ListValue,'preferred_controller','Preferred TellStick',
   {automatic:'Automatic',...Object.fromEntries((data.controllers||[]).filter(usb=>usb.serial).map(usb=>[usb.serial,(usb.product||'TellStick')+' ('+usb.serial+')']))},
   'Used for sending. If unplugged, the setting returns to Automatic. Reconnecting the stick does not restore the preference.');
  this.preferredOption=preferred;
  preferred.validate=(_sid,v)=>v==='automatic'||(fresh.controllers||[]).some(usb=>usb.serial===v)?true:'Select Automatic or a connected TellStick.';
  command(advanced,'_diagnostics','Troubleshooting','Show details',()=>{ui.showModal('Home Controller diagnostics',[E('h4',{},'Network listeners'),E('pre',{},JSON.stringify(fresh.access||{},null,2)),E('h4',{},'Recent commands'),E('pre',{},JSON.stringify(fresh.status||{},(k,v)=>k==='at'&&typeof v==='number'?new Date(v*1000).toISOString():v,2)),E('div',{class:'right'},E('button',{class:'cbi-button',click:ui.hideModal},'Close'))]);return '';});
  const disclaimer=m.section(form.NamedSection,'disclaimer','disclaimer','Experimental, AI-generated software. Use at your own risk.');
  disclaimer.render=()=>E('div',{'class':'cbi-section','data-tab':'disclaimer','data-tab-title':disclaimer.title},warning.page(this.disclaimer));
  const node=await m.render();node.append(service,E('link',{rel:'stylesheet',href:L.resource('home-controller-help.css')+'?v=025'}));
  this.disclaimerActive=!!node.querySelector('.cbi-map-tabbed > [data-tab="disclaimer"][data-tab-active="true"]');
  for(const pane of node.querySelectorAll('.cbi-map-tabbed > [data-tab]'))pane.addEventListener('cbi-tab-active',()=>{this.disclaimerActive=pane.dataset.tab==='disclaimer';if(this.footer)this.footer.hidden=this.disclaimerActive;});
  if(this.footer)this.footer.hidden=this.disclaimerActive;
  tell(this.saveMessage||(this.saveMessage=feedback()),data.pending?'Saved changes are waiting to be applied.':'');return node;
 },
 addFooter(){
  this.saveMessage ||= feedback();
  // The native LuCI action bar; this package stores its configuration outside UCI.
  this.footer=E('div',{hidden:this.map&&!this.disclaimerActive?null:''},[this.saveMessage,view.prototype.addFooter.call(this)]);return this.footer;
 },
 async saveConfiguration(activate){
  if(!this.map||this.saving)return;this.saving=true;tell(this.saveMessage,activate?'Applying…':'Saving…');
  try{await this.map.save(null,true);const c=configuration(this.map,this.original),usb=(this.connectedControllers||[]).find(d=>d.serial===c.preferred_controller),instance=usb?usb.bus+':'+usb.device:'';
   const response=result(await (activate?apply:stage)(JSON.stringify(c),instance));
   if(Object.hasOwn(response,'preferred_controller')){c.preferred_controller=response.preferred_controller;this.map.data.set('json','advanced','preferred_controller',c.preferred_controller||'automatic');this.preferredOption?.getUIElement('advanced')?.setValue(c.preferred_controller||'automatic');}
   this.original=c;tell(this.saveMessage,activate?'Settings applied.':'Saved. Use Save & Apply to activate these changes.');}
  catch(e){tell(this.saveMessage,e.message||String(e),true);}finally{this.saving=false;}
 },
 handleSave(){return this.saveConfiguration(false);},handleSaveApply(){return this.saveConfiguration(true);},
 async handleReset(){
  if(!this.map||this.saving)return;this.saving=true;
  try{result(await discard());const data=result(await get()),old=this.map.root;old.replaceWith(await this.renderSettings(data));tell(this.saveMessage,'Unapplied changes discarded.');}
  catch(e){tell(this.saveMessage,e.message||String(e),true);}finally{this.saving=false;}
 }
});
