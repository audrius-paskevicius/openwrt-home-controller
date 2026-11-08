'use strict';
'require view';
'require rpc';
'require ui';
const get = rpc.declare({object:'home-controller',method:'get',expect:{}});
const save = rpc.declare({object:'home-controller',method:'save',params:['config'],expect:{}});
const password = rpc.declare({object:'home-controller',method:'password',params:['user','password'],expect:{}});
const revoke = rpc.declare({object:'home-controller',method:'revoke',expect:{}});
const sessions = rpc.declare({object:'home-controller',method:'sessions',expect:{}});
const tls = rpc.declare({object:'home-controller',method:'tls',expect:{}});
function result(r){if(r.error)throw new Error(r.error);return r;}
function field(label,value,type){const input=E('input',{class:'cbi-input-text',type:type||'text',value:value==null?'':String(value)});return {input,node:E('label',{style:'display:block;margin:10px 0'},[E('span',{style:'display:block;font-weight:600'},label),input])};}
function select(label,value,choices){const input=E('select',{class:'cbi-input-select'},choices.map(x=>E('option',{value:x[0]},x[1])));input.value=value;return {input,node:E('label',{style:'display:block;margin:10px 0'},[E('span',{style:'display:block;font-weight:600'},label),input])};}
function notice(e){ui.addNotification(null,E('p',{},e.message||String(e)),'error');}
return view.extend({
 load:function(){return get().then(result);},
 render:function(data){
 const c=data.config, root=E('div',{},[E('h2',{},'Home Controller'),E('p',{},'Manage devices, household access and schedules. Household users can operate controls but cannot change these settings.')]);
 const general=E('div',{class:'cbi-section'},[E('h3',{},'Service and access')]);root.append(general);
 const enabled=select('Service',String(c.enabled),[['false','Disabled'],['true','Enabled']]);
 const port=field('HTTPS port',c.port,'number'),networks=field('Allowed logical networks (comma separated; active VPN tunnels are also detected)',c.networks.join(', '));
 const serial=field('TellStick serial (choose the replacement serial when swapping hardware)',c.controller);
 const zone=field('Time zone (router = follow router setting)',c.timezone),lat=field('Latitude',c.latitude,'number'),lon=field('Longitude',c.longitude,'number'),days=field('Remember household sessions for days',c.session_days,'number');lat.input.step=lon.input.step='any';
 [enabled,port,networks,serial,zone,lat,lon,days].forEach(x=>general.append(x.node));
 general.append(E('p',{},data.sunwait?'Sunwait is available.':'Solar schedules need the optional sunwait package.'));
 if(data.controllers)general.append(E('p',{},'Detected controllers: '+data.controllers.map(x=>x.product+' ('+x.serial+')').join(', ')));
 if(data.access)general.append(E('pre',{},JSON.stringify(data.access,null,2)));
 general.append(E('button',{class:'cbi-button',click:async()=>{try{result(await tls());ui.addNotification(null,E('p',{},'Router HTTPS certificate copied. Home Controller is restarting.'),'info');}catch(e){notice(e);}}},'Refresh HTTPS certificate from router'));
 const split=s=>s.split(',').map(x=>x.trim()).filter(Boolean);
 const tables={};
 function editor(key,title,columns,defaults){const section=E('div',{class:'cbi-section'},[E('h3',{},title)]),rows=E('div');tables[key]=[];
  function add(obj){const row=E('div',{style:'border:1px solid #8885;padding:14px;margin:12px 0;border-radius:8px'}),entry={};columns.forEach(col=>{let value=obj[col[0]];if(Array.isArray(value))value=value.join(', ');const f=col[2]==='select'?select(col[1],String(value),col[3]):field(col[1],value,col[2]);entry[col[0]]={field:f,col};row.append(f.node);});const remove=E('button',{class:'cbi-button cbi-button-remove',click:()=>{row.remove();tables[key]=tables[key].filter(x=>x!==entry);}},'Remove');row.append(remove);tables[key].push(entry);rows.append(row);}
  (c[key]||[]).forEach(add);section.append(rows,E('button',{class:'cbi-button cbi-button-add',click:()=>add(Object.assign({},defaults))},'Add'));root.append(section);
 }
 editor('devices','Devices',[['id','ID'],['name','Display name'],['radio_id','Telldus ID','number'],['house','Hasta house code','number'],['unit','Hasta unit (0–15)','number']],{id:'',name:'',radio_id:1,house:1,unit:1});
 editor('groups','Groups',[['id','ID'],['name','Display name'],['devices','Device IDs, comma separated']],{id:'',name:'',devices:[]});
 editor('schedules','Schedules',[['id','ID'],['enabled','Enabled','select',[['true','Yes'],['false','No']]],['target','Device or group ID'],['action','Action','select',[['up','Up'],['stop','Stop'],['down','Down']]],['days','Weekdays: 0 Sunday, 1 Monday, … 6 Saturday'],['event','Event','select',[['time','Fixed time'],['sunrise','Sunrise'],['sunset','Sunset'],['dawn','Civil dawn'],['dusk','Civil dusk']]],['time','Fixed time (HH:MM)'],['offset','Solar offset in minutes (negative = before)','number'],['earliest','Not before (HH:MM, optional)'],['latest','Not after (HH:MM, optional)'],['fallback','If solar event is absent (HH:MM, blank = skip)']],{id:'',enabled:false,target:'',action:'up',days:[1,2,3,4,5],event:'time',time:'08:00',offset:0,earliest:'',latest:'',fallback:''});
 root.append(E('p',{},'Manual commands act immediately; later scheduled actions stay active. Missed events are skipped. Stop takes priority over queued movement. All displayed radio results describe commands, not confirmed positions.'));
 const apply=E('button',{class:'cbi-button cbi-button-apply',click:async()=>{apply.disabled=true;try{const n=Object.assign({},c,{enabled:enabled.input.value==='true',port:Number(port.input.value),networks:split(networks.input.value),controller:serial.input.value.trim(),timezone:zone.input.value.trim(),latitude:Number(lat.input.value),longitude:Number(lon.input.value),session_days:Number(days.input.value)});for(const key of Object.keys(tables)){n[key]=tables[key].map(entry=>{const o={};for(const k of Object.keys(entry)){const x=entry[k];let v=x.field.input.value.trim();if(x.col[2]==='number')v=Number(v);if(k==='enabled')v=v==='true';if(k==='devices')v=split(v);if(k==='days')v=split(v).map(Number);o[k]=v;}return o;});}result(await save(JSON.stringify(n)));ui.addNotification(null,E('p',{},'Saved. Home Controller is restarting.'),'info');}catch(e){notice(e);}finally{apply.disabled=false;}}},'Save and apply');root.append(apply);
 const account=E('div',{class:'cbi-section'},[E('h3',{},'Household account')]),user=field('Username',data.account||'household'),pass=field('New password (at least 12 characters)','','password');pass.input.autocomplete='new-password';account.append(user.node,pass.node,E('button',{class:'cbi-button',click:async()=>{try{result(await password(user.input.value,pass.input.value));pass.input.value='';ui.addNotification(null,E('p',{},'Account updated; previous sessions revoked.'),'info');}catch(e){notice(e);}}},'Set household password'),E('button',{class:'cbi-button',style:'margin-left:10px',click:async()=>{try{result(await revoke());ui.addNotification(null,E('p',{},'All household sessions revoked.'),'info');}catch(e){notice(e);}}},'Sign out all devices'));
 const sessionList=E('pre');account.append(E('button',{class:'cbi-button',style:'margin-left:10px',click:async()=>{try{sessionList.textContent=JSON.stringify(result(await sessions()).sessions,null,2);}catch(e){notice(e);}}},'Show sessions'),sessionList);root.append(account);
 root.append(E('details',{},[E('summary',{},'Recent commands and service state'),E('pre',{},JSON.stringify(data.status||{},null,2))]));return root;
 },handleSaveApply:null,handleSave:null,handleReset:null
});
