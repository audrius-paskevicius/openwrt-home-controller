'use strict';
'require view';
'require rpc';
'require ui';
'require poll';
const get=rpc.declare({object:'home-controller',method:'get',expect:{}});
const save=rpc.declare({object:'home-controller',method:'save',params:['config'],expect:{}});
const password=rpc.declare({object:'home-controller',method:'password',params:['user','password'],expect:{}});
const revoke=rpc.declare({object:'home-controller',method:'revoke',expect:{}});
const sessions=rpc.declare({object:'home-controller',method:'sessions',expect:{}});
const tls=rpc.declare({object:'home-controller',method:'tls',expect:{}});
const clone=x=>JSON.parse(JSON.stringify(x));
const actionNames={up:'Up',stop:'Stop',down:'Down'},eventNames={time:'Fixed time',sunrise:'Sunrise',sunset:'Sunset',dawn:'Civil dawn',dusk:'Civil dusk'};
const weekdays=[[1,'Mon'],[2,'Tue'],[3,'Wed'],[4,'Thu'],[5,'Fri'],[6,'Sat'],[0,'Sun']];
const symbols=[['none','None'],['blind','Blind'],['clock','Clock'],['lamp','Lamp'],['plug','Plug'],['fan','Fan']];
let fieldId=0;
function result(r){if(r.error)throw Error(r.error);return r;}
function feedback(){const node=E('p',{role:'status','aria-live':'polite'});node.hidden=true;return node;}
function tell(node,text,error){node.textContent=text;node.hidden=!text;node.classList.toggle('alert-message',!!error);node.classList.toggle('error',!!error);}
function button(text,click,primary){return E('button',{type:'button',class:'cbi-button '+(primary?'cbi-button-apply':'cbi-button-neutral'),click},text);}
function valueRow(label,widget,input){input.id='hc-field-'+(++fieldId);return {input,node:E('div',{class:'cbi-value'},[E('label',{class:'cbi-value-title',for:input.id},label),E('div',{class:'cbi-value-field'},widget)])};}
function field(label,value,type){const widget=new ui.Textfield(value==null?'':String(value),{password:type==='password'}).render(),input=widget.querySelector('input');input.type=type||'text';return valueRow(label,widget,input);}
function select(label,value,choices){const widget=new ui.Select(String(value),Object.fromEntries(choices),{sort:choices.map(x=>String(x[0]))}).render(),input=widget.querySelector('select');return valueRow(label,widget,input);}
function check(label,value){const node=new ui.Checkbox(value?'1':'0',{}).render(),input=node.querySelector('input[type=checkbox]');node.querySelector('label').append(' '+label);return {input,node};}
function section(title){return E('div',{class:'cbi-section'},[E('h3',{},title)]);}
function spaced(nodes,gap=' '){return nodes.flatMap((node,i)=>i?[gap,node]:[node]);}
function hide(node,hidden){node.hidden=hidden;node.classList.toggle('hidden',hidden);}
function integer(f,min,max){const n=Number(f.input.value);if(f.input.value===''||!Number.isInteger(n)||n<min||n>max)throw Error(f.node.firstChild.textContent+': enter a number from '+min+' to '+max+'.');return n;}
return view.extend({
 load:function(){return get().then(result);},
 render:function(data){
  let c=clone(data.config),saving=false;
  const rowMessages=new Map();function rowFeedback(id){if(!rowMessages.has(id))rowMessages.set(id,feedback());return rowMessages.get(id);}
  const root=E('div',{class:'cbi-map hc-admin'},[E('h2',{},'Home Controller')]);
  const paneGroup=E('div'),panels={};root.append(paneGroup);
  const tabSpec=[['overview','Overview'],['devices','Devices & groups'],['schedules','Schedules'],['access','Household access'],['advanced','Advanced']];
  for(const [id,name] of tabSpec){panels[id]=E('div',{id:'hc-panel-'+id,'data-tab':id,'data-tab-title':name});paneGroup.append(panels[id]);}
  async function commit(next,control,message,done){if(saving){tell(message,'Another change is still saving.',true);return;}saving=true;control.disabled=true;tell(message,'Saving…');try{result(await save(JSON.stringify(next)));c=next;renderLists();if(done)done();tell(message,'Saved.');}catch(e){tell(message,e.message||String(e),true);}finally{saving=false;control.disabled=false;}}
  function uniqueId(prefix){let id;do{id=prefix+'_'+Date.now().toString(36)+'_'+Math.random().toString(36).slice(2,6);}while([...c.devices,...c.groups,...c.schedules].some(x=>x.id===id));return id;}
  function nameFor(id){return [...c.devices,...c.groups].find(x=>x.id===id)?.name||'Missing device';}
  function nextText(state){if(!state)return 'Waiting for service status';if(!state.running)return 'Stopped · manual controls remain available';if(!state.enabled_schedules)return 'Running · no schedules enabled';if(!state.next)return 'Running · no event in the next week';return state.next.target+' · '+actionNames[state.next.action]+' · '+new Date(state.next.at).toLocaleString([],{weekday:'short',hour:'2-digit',minute:'2-digit',timeZone:state.timezone});}
  const overview=panels.overview,metrics=section('Status'),serviceMetric=E('div',{class:'cbi-value'}),usbMetric=E('div',{class:'cbi-value'}),schedulerMetric=E('div',{class:'cbi-value'});metrics.append(serviceMetric,usbMetric,schedulerMetric);overview.append(metrics);
  const overviewMessage=feedback(),overviewActions=E('div',{class:'cbi-section'});overview.append(overviewActions,overviewMessage);
  const countSummary=E('p',{class:'cbi-section-descr'});overview.append(countSummary);
  function metric(node,title,content){node.replaceChildren(E('label',{class:'cbi-value-title'},title),E('div',{class:'cbi-value-field'},content));}
  function updateOverview(fresh){
   metric(serviceMetric,'Service',fresh.config.enabled?'Enabled':'Disabled');
   const controllers=fresh.controllers||[],active=fresh.active_controller;
   metric(usbMetric,'TellStick',controllers.length>1?(active?'Using '+active.product+' ('+active.serial+')':'Selected automatically when started'):(controllers.length?'Connected':'Not connected'));
   metric(schedulerMetric,'Scheduler',fresh.config.enabled?nextText(fresh.status?.scheduler):'Service is disabled');
   overviewActions.replaceChildren();const listener=fresh.access?.listeners?.[0];if(listener)overviewActions.append(E('a',{class:'cbi-button cbi-button-apply',href:'https://'+listener+'/',target:'_blank',rel:'noopener'},'Open household controls'));
   overviewActions.append(' ',button('Refresh status',async()=>{try{const fresh=result(await get());updateOverview(fresh);tell(overviewMessage,'Status updated.');}catch(e){tell(overviewMessage,e.message,true);}}));
   countSummary.textContent=c.devices.length+' devices · '+c.groups.length+' groups · '+c.schedules.length+' schedules';
  }
  updateOverview(data);
  poll.add(()=>{if(!root.isConnected)return;return get().then(result).then(updateOverview).catch(()=>tell(overviewMessage,'Unable to refresh service status.',true));},10);
  const deviceList=E('div'),groupList=E('div'),scheduleList=E('div');
  const deviceSection=section('Devices'),groupSection=section('Groups'),scheduleSection=section('Schedules');
  deviceSection.append(E('div',{class:'cbi-section-create'},[E('p',{class:'cbi-section-descr'},'Names and order used on the household page.'),button('Add device',()=>editDevice(),true)]),deviceList);
  groupSection.append(E('div',{class:'cbi-section-create'},[E('p',{class:'cbi-section-descr'},'Groups appear after individual devices.'),button('Add group',()=>editGroup(),true)]),groupList);
  panels.devices.append(deviceSection,groupSection);
  scheduleSection.append(E('div',{class:'cbi-section-create'},[E('p',{class:'cbi-section-descr'},'Manual controls do not change future schedules.'),button('Add schedule',()=>editSchedule(),true)]),scheduleList);
  panels.schedules.append(scheduleSection);
  function modal(title,fields,build,key,id){
   const message=feedback(),saveButton=button('Save changes',async()=>{try{const item=build(),next=clone(c),index=next[key].findIndex(x=>x.id===id);if(index<0)next[key].push(item);else next[key][index]=item;await commit(next,saveButton,message);}catch(e){tell(message,e.message,true);}},true);
   ui.showModal(title,[E('div',{class:'hc-editor'},fields),E('div',{class:'hc-editor'},[message,E('div',{class:'right'},spaced([saveButton,button('Close',()=>ui.hideModal())]))])]);
  }
  function removeItem(key,item){
   const references=[];if(key==='devices')for(const g of c.groups)if(g.devices.includes(item.id))references.push('group '+g.name);
   if(key!=='schedules')for(const s of c.schedules)if(s.target===item.id)references.push('a schedule for '+nameFor(item.id));
   const message=feedback(),cancel=button('Cancel',()=>ui.hideModal());let actions=[cancel];
   if(!references.length){const remove=button('Delete',async()=>{const next=clone(c);next[key]=next[key].filter(x=>x.id!==item.id);await commit(next,remove,message,()=>{remove.remove();cancel.textContent='Close';});});actions.push(remove);}
   ui.showModal('Delete '+(item.name||'schedule')+'?',[E('p',{},references.length?'Used by '+references.join(', ')+'. Edit those references first.':'This removes the configuration. It does not send a radio command.'),message,E('div',{class:'right'},spaced(actions))]);
  }
  function editDevice(existing){
   const used=new Set(c.devices.map(x=>x.radio_id));let radio=1;while(used.has(radio))radio++;
   const d=clone(existing||{id:uniqueId('device'),name:'',radio_id:radio,house:'',unit:1});
   const name=field('Name',d.name),icon=select('Symbol',d.icon||'blind',symbols),house=field('House code',d.house,'number'),unit=field('Unit',d.unit,'number');name.input.maxLength=80;house.input.min=1;house.input.max=65535;unit.input.min=0;unit.input.max=15;
   const details=E('details',existing?{}:{open:''},[E('summary',{},'Radio details'),E('p',{class:'cbi-section-descr'},'Hasta selflearningv2. These codes stay the same when replacing the TellStick.'),E('div',{class:'cbi-section-node'},[house.node,unit.node])]);
   modal(existing?'Edit device':'Add device',[name.node,icon.node,details],()=>{const value=name.input.value.trim();if(!value)throw Error('Enter a device name.');return Object.assign({},d,{name:value,icon:icon.input.value,house:integer(house,1,65535),unit:integer(unit,0,15)});},'devices',d.id);
  }
  function editGroup(existing){const g=clone(existing||{id:uniqueId('group'),name:'',devices:[]}),name=field('Name',g.name),icon=select('Symbol',g.icon||'blind',symbols);name.input.maxLength=80;
   const members=c.devices.map(d=>({id:d.id,...check(d.name,g.devices.includes(d.id))}));
   modal(existing?'Edit group':'Add group',[name.node,icon.node,E('p',{},'Include these devices'),E('div',{},spaced(members.map(x=>x.node)))],()=>{const value=name.input.value.trim(),devices=members.filter(x=>x.input.checked).map(x=>x.id);if(!value)throw Error('Enter a group name.');if(!devices.length)throw Error('Select at least one device.');return {id:g.id,name:value,devices,icon:icon.input.value};},'groups',g.id);
  }
  function editSchedule(existing){
   const s=clone(existing||{id:uniqueId('schedule'),enabled:false,target:c.devices[0]?.id||c.groups[0]?.id||'',action:'up',days:[1,2,3,4,5],event:'time',time:'08:00',offset:0,earliest:'',latest:'',fallback:''});
   const enabled=check('Enable this schedule',s.enabled),target=select('Device or group',s.target,[...c.devices.map(d=>[d.id,d.name]),...c.groups.map(g=>[g.id,g.name+' · group'])]),action=select('Action',s.action,Object.entries(actionNames)),event=select('When',s.event,Object.entries(eventNames)),time=field('Time',s.time,'time');
   const days=weekdays.map(([id,label])=>({id,...check(label,s.days.includes(id))}));
   const offset=field('Minutes before (−) or after (+)',s.offset,'number'),earliest=field('Not before (optional)',s.earliest,'time'),latest=field('Not after (optional)',s.latest,'time'),fallback=field('If the solar event does not occur (optional)',s.fallback,'time');
   offset.input.min=-720;offset.input.max=720;
   const solar=E('div',{},[offset.node,E('details',{},[E('summary',{},'Time limits and fallback'),earliest.node,latest.node,fallback.node]),E('p',{class:'cbi-section-descr'},data.sunwait?'Solar times use the configured location below.':'Install the sunwait package before enabling a solar schedule.')]);
   const change=()=>{hide(time.node,event.input.value!=='time');hide(solar,event.input.value==='time');};event.input.addEventListener('change',change);change();
   modal(existing?'Edit schedule':'Add schedule',[enabled.node,E('div',{},[target.node,action.node]),E('p',{},'Repeat on'),E('div',{class:'cbi-value-field'},spaced(days.map(x=>x.node),'\u00a0\u00a0')),event.node,time.node,solar],()=>{
    const chosen=days.filter(x=>x.input.checked).map(x=>x.id);if(!target.input.value)throw Error('Add a device or group first.');if(!chosen.length)throw Error('Choose at least one weekday.');
    const isSolar=event.input.value!=='time';if(!isSolar&&!time.input.value)throw Error('Choose a time.');if(isSolar&&enabled.input.checked&&!data.sunwait)throw Error('Install sunwait before enabling this solar schedule.');if(earliest.input.value&&latest.input.value&&earliest.input.value>latest.input.value)throw Error('The earliest time must not be after the latest time.');
    return Object.assign({},s,{enabled:enabled.input.checked,target:target.input.value,action:action.input.value,days:chosen,event:event.input.value,time:isSolar?'':time.input.value,offset:isSolar?integer(offset,-720,720):0,earliest:isSolar?earliest.input.value:'',latest:isSolar?latest.input.value:'',fallback:isSolar?fallback.input.value:''});
   },'schedules',s.id);
  }
  function row(title,description,actions,message){const copy=E('td',{class:'td'},[E('strong',{},title),E('br'),E('small',{},description)]);if(message)copy.append(message);return E('tr',{class:'tr cbi-section-table-row hc-row'},[copy,E('td',{class:'td cbi-section-actions'},E('div',{},spaced(actions)))]);}
  function table(rows){return E('table',{class:'table cbi-section-table'},[E('thead',{},E('tr',{class:'tr'},[E('th',{class:'th'},'Name / details'),E('th',{class:'th cbi-section-actions'},'Actions')])),E('tbody',{},rows)]);}
  function renderLists(){
   deviceList.replaceChildren(table(c.devices.map((d,index)=>{const message=rowFeedback('device:'+d.id),up=button('Move up',async()=>{const next=clone(c);[next.devices[index-1],next.devices[index]]=[next.devices[index],next.devices[index-1]];await commit(next,up,message);});up.disabled=index===0;return row(d.name,'Individual control',[button('Edit',()=>editDevice(d)),up,button('Delete',()=>removeItem('devices',d))],message);} )));
   if(!c.devices.length)deviceList.append(E('p',{class:'cbi-section-descr'},'No devices yet. Add your first device.'));
   groupList.replaceChildren(table(c.groups.map(g=>row(g.name,g.devices.map(nameFor).join(', '),[button('Edit',()=>editGroup(g)),button('Delete',()=>removeItem('groups',g))]))));if(!c.groups.length)groupList.append(E('p',{class:'cbi-section-descr'},'No groups yet.'));
   scheduleList.replaceChildren(table(c.schedules.map(s=>{const enabled=check('Enabled',s.enabled),message=rowFeedback('schedule:'+s.id);enabled.input.addEventListener('change',async()=>{const next=clone(c);next.schedules.find(x=>x.id===s.id).enabled=enabled.input.checked;await commit(next,enabled.input,message);enabled.input.checked=c.schedules.find(x=>x.id===s.id)?.enabled;});const when=s.event==='time'?s.time:eventNames[s.event]+(s.offset?' '+(s.offset>0?'+':'−')+' '+Math.abs(s.offset)+' min':'');const days=weekdays.filter(([id])=>s.days.includes(id)).map(([,name])=>name).join(', ');return row(nameFor(s.target)+' · '+actionNames[s.action],when+' · '+days,[enabled.node,button('Edit',()=>editSchedule(s)),button('Delete',()=>removeItem('schedules',s))],message);})));if(!c.schedules.length)scheduleList.append(E('p',{class:'cbi-section-descr'},'No schedules yet. Manual controls are always available while the service is enabled.'));
   countSummary.textContent=c.devices.length+' devices · '+c.groups.length+' groups · '+c.schedules.length+' schedules';
  }
  renderLists();
  const locationPanel=E('details',{class:'cbi-section'},[E('summary',{},'Location and time zone for schedules')]);
  const lat=field('Latitude',c.latitude,'number'),lon=field('Longitude',c.longitude,'number'),zoneMode=select('Time zone',c.timezone==='router'?'router':'custom',[['router','Follow router automatically'],['custom','Use a different time zone']]),zone=field('Time zone name',c.timezone==='router'?'':c.timezone);
  lat.input.step=lon.input.step='any';hide(zone.node,zoneMode.input.value==='router');zoneMode.input.addEventListener('change',()=>hide(zone.node,zoneMode.input.value==='router'));
  const locationMessage=feedback(),saveLocation=button('Save location',async()=>{if(lat.input.value===''||lon.input.value===''){tell(locationMessage,'Enter latitude and longitude.',true);return;}const next=clone(c);Object.assign(next,{latitude:Number(lat.input.value),longitude:Number(lon.input.value),timezone:zoneMode.input.value==='router'?'router':zone.input.value.trim()});await commit(next,saveLocation,locationMessage);},true);
  locationPanel.append(E('p',{class:'cbi-section-descr'},'Set once for sunrise and sunset schedules. Fixed-time schedules only need the time zone.'),E('div',{class:'cbi-section-node'},[lat.node,lon.node]),zoneMode.node,zone.node,saveLocation,locationMessage);panels.schedules.append(locationPanel);
  const account=E('div',{class:'cbi-section'},[E('h3',{},'Household account')]),user=field('Username',data.account||'household'),pass=field('New password','','password');pass.input.autocomplete='new-password';
  const accountFeedback=feedback();
  const setPassword=button('Set household password',async()=>{if(setPassword.disabled)return;if(!pass.input.value){tell(accountFeedback,'Enter a new password.',true);pass.input.focus();return;}if([...pass.input.value].length>256){tell(accountFeedback,'Password must be at most 256 characters.',true);return;}const username=user.input.value.trim();if(!/^[a-z][a-z0-9_-]{0,31}$/.test(username)){tell(accountFeedback,'Use a username starting with a lowercase letter, then lowercase letters, numbers, hyphens or underscores (up to 32 characters).',true);return;}setPassword.disabled=true;tell(accountFeedback,'Saving password…');try{result(await password(username,pass.input.value));pass.input.value='';tell(accountFeedback,'Password changed. Previous sessions have been signed out.');}catch(e){tell(accountFeedback,e.message||String(e),true);}finally{setPassword.disabled=false;}},true);
  pass.input.addEventListener('keydown',event=>{if(event.key==='Enter'){event.preventDefault();setPassword.click();}});account.append(E('div',{class:'cbi-section-node'},[user.node,pass.node]),setPassword,accountFeedback);panels.access.append(account);
  const display=section('Household display'),schedulerSymbol=select('Scheduler symbol',c.scheduler_icon||'clock',symbols),displayMessage=feedback();
  const saveDisplay=button('Save symbol',async()=>{const next=clone(c);next.scheduler_icon=schedulerSymbol.input.value;await commit(next,saveDisplay,displayMessage);},true);
  display.append(schedulerSymbol.node,E('p',{class:'cbi-section-descr'},'Device and group symbols are selected in their editors. Symbols only change appearance.'),saveDisplay,displayMessage);panels.access.append(display);
  const sessionSection=section('Remembered sessions'),days=field('Remember sign-in for days',c.session_days,'number'),sessionFeedback=feedback();days.input.min=1;days.input.max=90;
  const saveDays=button('Save duration',async()=>{try{const next=clone(c);next.session_days=integer(days,1,90);await commit(next,saveDays,sessionFeedback);}catch(e){tell(sessionFeedback,e.message,true);}});
  const sessionList=E('div');sessionSection.append(days.node,E('div',{},spaced([saveDays,button('Show signed-in devices',async()=>{try{const items=result(await sessions()).sessions;sessionList.replaceChildren(table(items.map(s=>row(s.device||'Browser','Signed in '+new Date(s.created*1000).toLocaleString()+' · expires '+new Date(s.expires*1000).toLocaleString(),[]))));if(!items.length)sessionList.append(E('p',{},'No active sessions.'));tell(sessionFeedback,'Sessions updated.');}catch(e){tell(sessionFeedback,e.message,true);}}),button('Sign out all devices',async()=>{try{result(await revoke());sessionList.replaceChildren();tell(sessionFeedback,'All household sessions have been signed out.');}catch(e){tell(sessionFeedback,e.message,true);}})])),sessionFeedback,sessionList);panels.access.append(sessionSection);
  const service=section('Service'),enabled=check('Enable Home Controller',c.enabled),serviceMessage=feedback();
  const serviceSave=button('Apply service setting',async()=>{const next=clone(c);next.enabled=enabled.input.checked;await commit(next,serviceSave,serviceMessage);},true);
  service.append(enabled.node,E('p',{class:'cbi-section-descr'},'Disabling the service also closes the household control page. To stop only scheduled actions, use Scheduler on that page.'),serviceSave,serviceMessage);panels.advanced.append(service);
  const network=E('details',{class:'cbi-section'},[E('summary',{},'Network overrides')]),port=field('HTTPS port',c.port,'number'),networkMessage=feedback();
  const networkNames=[...new Set(['lan',...(data.networks||[]),...c.networks])],choices=networkNames.map(name=>({name,...check(name,c.networks.includes(name))}));
  const networkSave=button('Save network settings',async()=>{try{const next=clone(c);next.port=integer(port,1024,65535);next.networks=choices.filter(x=>x.input.checked).map(x=>x.name);await commit(next,networkSave,networkMessage);}catch(e){tell(networkMessage,e.message,true);}});
  network.append(E('p',{class:'cbi-section-descr'},'Default access is LAN plus detected VPN tunnels. WAN is excluded. Existing firewall restrictions still apply.'),port.node,E('div',{class:'cbi-value-field'},choices.map(x=>x.node)),networkSave,networkMessage);panels.advanced.append(network);
  const https=E('details',{class:'cbi-section'},[E('summary',{},'HTTPS certificate')]),tlsMessage=feedback(),tlsButton=button('Renew certificate',async()=>{tlsButton.disabled=true;tell(tlsMessage,'Renewing certificate…');try{result(await tls());tell(tlsMessage,'Certificate renewed. The household service is restarting.');}catch(e){tell(tlsMessage,e.message,true);}finally{tlsButton.disabled=false;}});https.append(tlsButton,tlsMessage);panels.advanced.append(https);
  const diagnostics=E('details',{class:'cbi-section'},[E('summary',{},'Troubleshooting'),E('h4',{},'Network listeners'),E('pre',{},JSON.stringify(data.access||{},null,2)),E('h4',{},'Recent commands'),E('pre',{},JSON.stringify(data.status||{},null,2))]);panels.advanced.append(diagnostics);
  ui.tabs.initTabGroup(Object.values(panels));return root;
 },handleSaveApply:null,handleSave:null,handleReset:null
});
