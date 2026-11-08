const assert = require('node:assert/strict');
const fs = require('node:fs');

// Exercise the actual LuCI event handlers and their inline feedback without a router.
class Element {
 constructor(tag, attrs = {}, children = []) {
  this.tag = tag; this.attrs = attrs; this.children = []; this.style = {};
  this.value = attrs.value || ''; this.disabled = false; this.events = {};
  this.classList = {toggle:()=>{}};
  this.append(...(Array.isArray(children) ? children : [children]));
 }
 append(...children) { this.children.push(...children); }
 replaceChildren(...children) { this.children = children; }
 setAttribute(key, value) { this.attrs[key] = value; }
 get firstChild() { return this.children[0]; }
 get textContent() { return this.children.map(c => typeof c === 'string' ? c : c.textContent).join(''); }
 set textContent(text) { this.children = [text]; }
 addEventListener(name, handler) { this.events[name] = handler; }
 focus() { this.focused = true; }
 click() { return this.attrs.click?.(); }
}
function descendants(e) { return [e, ...e.children.filter(c => c instanceof Element).flatMap(descendants)]; }

(async () => {
 const calls = [];
 let respond = () => Promise.resolve({ok:true});
 const rpc = {declare: spec => (...args) => { calls.push({method:spec.method,args}); return respond(); }};
 const view = new Function('view', 'rpc', 'ui', 'E', 'poll', fs.readFileSync('luci/home-controller.js','utf8'))(
  {extend:x=>x}, rpc, {addNotification:()=>assert.fail('Account feedback must be inline')},
  (tag, attrs, children) => new Element(tag, attrs, children), {add:()=>{}});
 const root = view.render({config:JSON.parse(fs.readFileSync('files/config.json')), account:'household', controllers:[]});
 const account = descendants(root).find(e=>e.tag==='div' && e.children[0]?.textContent==='Household account');
 const nodes = descendants(account), inputs = nodes.filter(e=>e.tag==='input');
 const button = nodes.find(e=>e.tag==='button' && e.textContent==='Set household password');
 const feedback = nodes.find(e=>e.attrs.role==='status');
 const pass = inputs.find(e=>e.attrs.type==='password');
 await button.click();
 assert.equal(calls.length,0); assert.match(feedback.textContent,/Enter a new password/);
 pass.value='short';
 respond=()=>Promise.resolve({error:'Server rejected this request'});
 await button.click();
 assert.equal(calls.at(-1).args[1],'short'); assert.match(feedback.textContent,/Server rejected/);
 assert.equal(pass.value,'short'); assert.equal(button.disabled,false);
 respond=()=>Promise.reject(new Error('Connection failed'));
 await button.click(); assert.match(feedback.textContent,/Connection failed/);
 let complete;
 respond=()=>new Promise(resolve=>{complete=resolve;});
 const saving=button.click(), count=calls.length;
 assert.equal(button.disabled,true); assert.match(feedback.textContent,/Saving password/);
 await button.click(); assert.equal(calls.length,count);
 complete({ok:true}); await saving;
 assert.equal(button.disabled,false); assert.equal(pass.value,'');
 assert.match(feedback.textContent,/Password changed/);
 pass.value='x'.repeat(257);
 await button.click(); assert.equal(calls.length,count); assert.match(feedback.textContent,/256/);
 console.log('PASS: short password submitted; inline empty/error/network/pending/success feedback; duplicate submissions blocked');
})().catch(e=>{console.error(e);process.exit(1);});
