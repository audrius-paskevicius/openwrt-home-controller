'use strict';
let csrf = '', signedIn = false, refreshing = false;
const $ = id => document.getElementById(id);
function message(text) { $('message').textContent = text; }
async function api(path, body) {
 const options = { credentials: 'same-origin', cache: 'no-store', signal: AbortSignal.timeout(15000) };
 if (body !== undefined) Object.assign(options, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-Home-Controller': '1', 'X-CSRF-Token': csrf }, body: JSON.stringify(body) });
 const response = await fetch('/api/' + path, options);
 const data = await response.json();
 if (!response.ok) { if (response.status === 401) showLogin(); throw new Error(data.error || 'Request failed'); }
 return data;
}
function showLogin() { signedIn = false; csrf = ''; $('login').hidden = false; $('controls').hidden = true; $('logout').hidden = true; }
function card(item) {
 const el = document.createElement('article'); el.className = 'card'; el.dataset.id = item.id;
 const title = document.createElement('h2'); title.textContent = item.name;
 const status = document.createElement('p'); status.className = 'status'; status.textContent = 'Position unknown';
 const buttons = document.createElement('div'); buttons.className = 'buttons';
 for (const action of ['up', 'stop', 'down']) {
  const button = document.createElement('button'); button.textContent = {up:'↑ Up',stop:'■ Stop',down:'↓ Down'}[action]; button.className = action;
  button.setAttribute('aria-label', item.name + ': ' + action);
  button.onclick = async () => { try { await api('command', {target:item.id,action}); status.textContent = 'Queued: ' + action; message(''); setTimeout(refresh,600); } catch(e) { message(e.message); } };
  buttons.append(button);
 }
 el.append(title,status,buttons); return el;
}
async function refresh() {
 if (refreshing) return; refreshing = true;
 try { const data = await api('state'); csrf = data.csrf;
  if (!signedIn) { $('devices').replaceChildren(...data.devices.map(card)); $('groups').replaceChildren(...(data.groups || []).map(card)); }
  signedIn = true; $('login').hidden = true; $('controls').hidden = false; $('logout').hidden = false;
  for (const d of data.devices) { const el = [...$('devices').children].find(x=>x.dataset.id===d.id); if(el) el.querySelector('.status').textContent = d.state.status ? d.state.action.toUpperCase() + ' · ' + d.state.status : 'Position unknown'; }
 } catch(e) { if(signedIn) message('Connection interrupted. ' + e.message); }
 finally { refreshing=false; }
}
$('login').onsubmit = async event => {event.preventDefault();const button=event.target.querySelector('button');button.disabled=true;try{const data=await api('login',{user:$('username').value,password:$('password').value,remember:$('remember').checked});csrf=data.csrf;$('password').value='';message('');await refresh();}catch(e){message(e.message);}finally{button.disabled=false;}};
$('logout').onclick = async () => {try{await api('logout',{});showLogin();message('Signed out.');}catch(e){message(e.message);}};
document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh();});setInterval(()=>{if(signedIn&&!document.hidden)refresh();},5000);refresh();
