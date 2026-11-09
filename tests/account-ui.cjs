const assert=require('node:assert/strict'),fs=require('node:fs');
// Exercise our form model and handlers. Native LuCI rendering is also checked on
// the target router with synthetic RPCs; these stubs do not imitate its CSS.
class Node {
 constructor(tag,attrs={},children=[]){this.tag=tag;this.attrs=attrs;this.children=[];this.classList={toggle(){}};this.append(...(Array.isArray(children)?children:[children]));}
 append(...items){this.children.push(...items);}replaceChildren(...items){this.children=items;}
 set textContent(v){this.children=[v];}get textContent(){return this.children.map(x=>typeof x==='string'?x:x?.textContent||'').join('');}
 replaceWith(){}querySelector(){return null;}querySelectorAll(){return [];}
}
const E=(...args)=>new Node(...args);
class Option {
 constructor(section,name,title){this.section=section;this.map=section.map;this.option=name;this.title=title;this.keylist=[];}
 value(k){this.keylist.push(k);return this;}depends(){}cfgvalue(sid){return this.map.data.get('json',sid,this.option);}
 formvalue(sid){return this.inputValue??this.cfgvalue(sid);}
 getUIElement(){return {setValue:v=>{this.inputValue=v;}};}renderWidget(){return E('button',{},this.inputtitle);}
}
class Section {
 constructor(map,type){this.map=map;this.type=type;this.children=[];}
 option(Type,name,title,...args){const o=new Type(this,name,title);if(Type===SectionValue)o.subsection=new Section(this.map,args[0]);this.children.push(o);return o;}
 getOption(name){return this.children.find(o=>o.option===name);}
}
class SectionValue extends Option {}
class JSONMap {
 constructor(model){const records=[];for(const [type,values] of Object.entries(model))for(const [i,v] of (Array.isArray(values)?values:[values]).entries())records.push({...v,'.type':type,'.name':v['.name']||(Array.isArray(values)?type+i:type)});
  this.data={sections:(_c,t)=>records.filter(r=>r['.type']===t),get:(_c,s,o)=>{const r=records.find(r=>r['.name']===s);return o?(r?.[o]==null?null:String(r[o])):r;},set:(_c,s,o,v)=>{records.find(r=>r['.name']===s)[o]=v;}};this.children=[];
 }
 section(Type,id,...args){const s=new Section(this,Type===Section?args[0]:id);this.children.push(s);return s;}
 async render(){return this.root=E('div');}async save(){}
}
const form={JSONMap,NamedSection:Section,GridSection:class extends Section{},Value:Option,ListValue:Option,MultiValue:Option,Flag:Option,DummyValue:Option,Button:Option,SectionValue};
(async()=>{
 process.env.TZ='Europe/Stockholm';
 const calls=[];let respond=()=>Promise.resolve({ok:true});
 const config=JSON.parse(fs.readFileSync('files/config.json'));config.enabled=true;config.devices=[{id:'sample',name:'Blind',radio_id:1,house:1234,unit:0}];config.groups=[{id:'all',name:'All',devices:['sample']}];
 const data={config,account:'home',controllers:[],sunwait:true,networks:['lan','wg_test','wan','wan6'],status:{scheduler:{running:true,enabled_schedules:1,timezone:'America/New_York',next:{target:'Blind',action:'up',at:'2026-11-08T01:00:00Z'}}}};
 let currentData=data;const polls=[];
 const rpc={declare:spec=>(...args)=>{calls.push({method:spec.method,args});return spec.method==='get'?Promise.resolve(currentData):respond();}};
 let diagnostic;
 const view=new Function('view','rpc','ui','form','poll','warning','E','L',fs.readFileSync('luci/home-controller.js','utf8'))({extend:x=>x},rpc,{showModal:(_title,body)=>{diagnostic=E('div',{},body);}},form,{add(fn){polls.push(fn);}},{gate:(_t,f)=>f()},E,{resource:x=>x});
 await view.render(data);
 assert.equal(view.map.children.find(s=>s.type==='general').getOption('_scheduler').renderWidget().textContent,'Running · Blind · Up · 2026-11-07 20:00');
 const networks=view.map.children.find(s=>s.type==='access').getOption('networks');
 assert.equal(networks.title,'Allowed networks');assert.deepEqual(networks.keylist,['lan','wg_test']);
 assert.notEqual(networks.validate('access',[]),true);assert.equal(networks.validate('access',['lan','wg_test']),true);
 const access=view.map.children.find(s=>s.type==='access'),pass=access.getOption('_password'),button=access.getOption('_password_action');
 assert.deepEqual(access.children.slice(0,3).map(o=>o.option),['port','networks','_tls']);
 view.map.data.set('json','access','port','9443');await view.handleSaveApply();
 assert.equal(JSON.parse(calls.at(-1).args[0]).port,9443);calls.length=0;
 const message=button.renderWidget().children[1],event={currentTarget:{disabled:true}}; // LuCI disables before invoking onclick.
 await button.onclick(event);assert.match(message.textContent,/Enter a new password/);assert.equal(calls.length,0);
 pass.inputValue='short';respond=()=>Promise.resolve({error:'Server rejected this request'});
 await button.onclick(event);assert.equal(calls.at(-1).args[1],'short');assert.match(message.textContent,/Server rejected/);assert.equal(pass.inputValue,'short');
 respond=()=>Promise.reject(Error('Connection failed'));await button.onclick(event);assert.match(message.textContent,/Connection failed/);
 let complete;respond=()=>new Promise(r=>{complete=r;});const pending=button.onclick(event),count=calls.length;
 await button.onclick(event);assert.equal(calls.length,count);assert.equal(event.currentTarget.disabled,true);
 complete({ok:true});await pending;assert.equal(pass.inputValue,'');assert.match(message.textContent,/Password changed/);
 pass.inputValue='x'.repeat(257);await button.onclick(event);assert.equal(calls.length,count);assert.match(message.textContent,/256/);
 respond=()=>Promise.resolve({ok:true});pass.inputValue='';
 view.map.data.set('json','access','session_days','7');await view.handleSave();assert.equal(calls.at(-1).method,'stage');
 let saved=JSON.parse(calls.at(-1).args[0]);assert.equal(saved.session_days,7);assert.deepEqual(saved.control_order,['device:sample','scheduler','group:all']);assert.equal(saved.groups[0].devices[0],'sample');assert.equal(saved.devices[0].unit,0);
 assert.ok(!JSON.stringify(saved).includes('_password'));assert.ok(!JSON.stringify(saved).includes('short'));
 await view.handleSaveApply();assert.equal(calls.at(-1).method,'save');
 // A VPN checkbox must survive Save & Apply, and an explicit deselection must too.
 view.map.data.set('json','access','networks',['wg_test']);await view.handleSaveApply();
 assert.deepEqual(JSON.parse(calls.at(-1).args[0]).networks,['wg_test']);
 view.map.data.set('json','access','networks',['lan']);await view.handleSaveApply();
 assert.deepEqual(JSON.parse(calls.at(-1).args[0]).networks,['lan']);
 respond=()=>Promise.resolve({error:'Synthetic save failure'});await view.handleSave();assert.match(view.saveMessage.textContent,/Synthetic save failure/);
 respond=()=>Promise.resolve({ok:true});await view.handleReset();assert.equal(calls.at(-2).method,'discard');assert.equal(view.map.data.get('json','access','session_days'),'30');
 respond=()=>Promise.resolve({sessions:[{device:'Test browser',created:Date.parse('2026-11-08T23:00:00Z')/1000,expires:Date.parse('2026-11-09T11:00:00Z')/1000}]});
 const updatedAccess=view.map.children.find(s=>s.type==='access');await updatedAccess.getOption('_sessions').onclick({currentTarget:{disabled:false}});
 const table=updatedAccess.getOption('_session_list').renderWidget().textContent;
 assert.ok(table.includes('2026-11-09 00:00'));assert.ok(table.includes('2026-11-09 12:00'));
 data.status.command={at:Date.parse('2026-11-08T21:10:12Z')/1000};
 await view.map.children.find(s=>s.type==='advanced').getOption('_diagnostics').onclick({currentTarget:{disabled:false}});
 assert.ok(diagnostic.textContent.includes('2026-11-08T21:10:12.000Z'),'diagnostics retain precise ISO timestamps');
 const a={serial:'TEST_A',product:'TellStick',capability:'Transmit only',bus:'1',device:'2'},b={serial:'TEST_B',product:'TellStick Duo',capability:'Transmit and receive',bus:'1',device:'3'};
 currentData={...data,controllers:[a,b],active_controller:a,config:{...config,preferred_controller:'TEST_A'}};
 await view.renderSettings(currentData);view.map.root.isConnected=true;
 const general=view.map.children.find(s=>s.type==='general'),pref=view.map.children.find(s=>s.type==='advanced').getOption('preferred_controller');
 assert.equal(general.getOption('_service'),undefined,'no duplicate Service Enabled row');
 const inventory=()=>general.getOption('_controller').renderWidget().textContent;
 assert.match(inventory(),/TEST_A/);assert.match(inventory(),/TEST_B/);assert.match(inventory(),/Transmit and receive/);assert.match(inventory(),/In use/);
 assert.deepEqual(pref.keylist,['automatic','TEST_A','TEST_B']);
 pref.inputValue='TEST_B';view.map.data.set('json','advanced','preferred_controller','TEST_B');
 respond=()=>Promise.resolve({ok:true,preferred_controller:'TEST_B'});await view.handleSaveApply();
 assert.equal(JSON.parse(calls.at(-1).args[0]).preferred_controller,'TEST_B');assert.equal(calls.at(-1).args[1],'1:3');
 currentData={...currentData,controllers:[a],config:{...config,preferred_controller:''}};await polls.at(-1)();
 assert.equal(pref.inputValue,'automatic');assert.deepEqual(pref.keylist,['automatic','TEST_A']);assert.ok(!inventory().includes('TEST_B'));
 currentData={...currentData,controllers:[a,b]};await polls.at(-1)();assert.equal(pref.inputValue,'automatic');assert.deepEqual(pref.keylist,['automatic','TEST_A','TEST_B']);
 currentData={...currentData,controllers:[],active_controller:null};await polls.at(-1)();assert.deepEqual(pref.keylist,['automatic']);assert.match(inventory(),/No TellStick connected/);
 const typed=JSON.parse(JSON.stringify(config));
 typed.devices.push({id:'lamp',name:'Test lamp',type:'switch',protocol:'arctech',house_code:'C',house:0,unit:7,radio_id:2});
 typed.groups.push({id:'lights',name:'Lights',devices:['lamp']});
 typed.devices.push({id:'dimmer',name:'Dimmer',type:'dimmer',protocol:'arctech',house_code:'D',house:0,unit:4,radio_id:3});
 typed.groups.push({id:'dimmers',name:'Dimmers',devices:['dimmer']});
 typed.schedules=[{id:'night',enabled:true,target:'lights',action:'off',days:[1],event:'time',time:'22:00',offset:0,earliest:'',latest:'',fallback:''}];
 await view.renderSettings({...data,config:typed});
 const rows=view.map.data.sections('json','control'),controlForm=view.map.children.find(s=>s.type==='control');
 assert.equal(rows.filter(x=>x.kind==='scheduler').length,1);assert.equal(rows.find(x=>x.id==='sample').kind,'blind');assert.equal(rows.find(x=>x.id==='lamp').kind,'switch');
 assert.ok(controlForm.getOption('kind').keylist.includes('dimmer'));
 const dimmerRow=rows.find(x=>x.id==='dimmer')['.name'];assert.equal(controlForm.getOption('unit').validate(dimmerRow,'16'),true);assert.notEqual(controlForm.getOption('unit').validate(dimmerRow,'0'),true);
 assert.notEqual(controlForm.getOption('devices').validate('', ['sample','lamp']),true);assert.equal(controlForm.getOption('devices').validate('', ['lamp']),true);
 const lampRow=rows.find(x=>x.id==='lamp')['.name'];assert.equal(controlForm.getOption('unit').validate(lampRow,'16'),true);assert.notEqual(controlForm.getOption('unit').validate(lampRow,'0'),true);
 const scheduleForm=view.map.children.find(s=>s.type==='scheduling').getOption('_schedules').subsection;
 assert.deepEqual(scheduleForm.getOption('days').keylist,['1','2','3','4','5','6','0']);
 const dayOption=scheduleForm.getOption('days'),getDays=dayOption.cfgvalue;dayOption.cfgvalue=()=>['0','6','1'];assert.equal(dayOption.textvalue('schedule_0'),'Mon, Sat, Sun');dayOption.cfgvalue=getDays;
 scheduleForm.addModalOptions(scheduleForm,'schedule_0');assert.deepEqual(scheduleForm.getOption('action').keylist,['on','off']);
 assert.ok(scheduleForm.getOption('target').keylist.includes('dimmer'));assert.ok(scheduleForm.getOption('target').keylist.includes('dimmers'));
 scheduleForm.getOption('target').inputValue='dimmers';scheduleForm.getOption('target').onchange({},'schedule_0','dimmers');assert.deepEqual(scheduleForm.getOption('action').keylist,['toggle','dim']);
 scheduleForm.getOption('target').inputValue='sample';scheduleForm.getOption('target').onchange({},'schedule_0','sample');
 assert.deepEqual(scheduleForm.getOption('action').keylist,['up','stop','down']);assert.equal(scheduleForm.getOption('action').inputValue,'up');
 respond=()=>Promise.resolve({ok:true});await view.handleSave();saved=JSON.parse(calls.at(-1).args[0]);
 assert.equal(saved.devices[0].type,'blind');assert.equal(saved.devices[1].type,'switch');assert.equal(saved.devices[1].house_code,'C');assert.equal(saved.devices[1].protocol,'arctech');assert.equal(saved.schedules[0].action,'off');assert.equal(saved.control_order.filter(x=>x==='scheduler').length,1);
 assert.equal(saved.devices[2].type,'dimmer');assert.equal(saved.devices[2].house_code,'D');assert.equal(saved.devices[2].house,0);assert.equal(saved.devices[2].protocol,'arctech');assert.ok(saved.control_order.includes('device:dimmer'));
 console.log('PASS: native-form handlers, Save/Apply/Reset, credential exclusion, current controller list/capabilities, preference submission, unplug reset and no preference restoration');
})().catch(e=>{console.error(e);process.exitCode=1;});
