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
 constructor(section,name,title){this.map=section.map;this.option=name;this.title=title;this.keylist=[];}
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
 const calls=[];let respond=()=>Promise.resolve({ok:true});
 const config=JSON.parse(fs.readFileSync('files/config.json'));config.devices=[{id:'sample',name:'Blind',radio_id:1,house:1234,unit:0}];config.groups=[{id:'all',name:'All',devices:['sample']}];
 const data={config,account:'home',controllers:[],sunwait:true};
 const rpc={declare:spec=>(...args)=>{calls.push({method:spec.method,args});return spec.method==='get'?Promise.resolve(data):respond();}};
 const view=new Function('view','rpc','ui','form','poll','warning','E',fs.readFileSync('luci/home-controller.js','utf8'))({extend:x=>x},rpc,{},form,{add(){}},{gate:(_t,f)=>f()},E);
 await view.render(data);
 const access=view.map.children.find(s=>s.type==='access'),pass=access.getOption('_password'),button=access.getOption('_password_action');
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
 respond=()=>Promise.resolve({error:'Synthetic save failure'});await view.handleSave();assert.match(view.saveMessage.textContent,/Synthetic save failure/);
 respond=()=>Promise.resolve({ok:true});await view.handleReset();assert.equal(calls.at(-2).method,'discard');assert.equal(view.map.data.get('json','access','session_days'),'30');
 console.log('PASS: native-form handlers, password feedback and duplicate guard, Save/Apply/Reset, configuration conversion and credential exclusion');
})().catch(e=>{console.error(e);process.exitCode=1;});
