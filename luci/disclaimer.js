'use strict';
'require request';
'require ui';

// Both interfaces load copies of the canonical DISCLAIMER.txt installed by the
// package. Remember the exact text, independently of the package version.
const key='home-controller-disclaimer';
let acknowledgedText=null;
function accepted(text){
 if(acknowledgedText===text)return true;
 try{return window.localStorage.getItem(key)===text;}catch(e){return false;}
}
function fullText(text){return E('div',{},text.trim().split(/\n\s*\n/).map((p,i)=>E(i?'p':'h3',{},p)));}
return L.Class.extend({
 page:function(text){return E('div',{},[
  E('link',{rel:'stylesheet',href:L.resource('home-controller-disclaimer.css')+'?v=023'}),
  E('h2',{},'Home Controller: disclaimer'),
  E('section',{class:'hc-disclaimer-notice'},fullText(text))
 ]);},
 load:function(){return request.get(L.resource('home-controller-disclaimer.txt'),{cache:false,timeout:10000}).then(response=>{
  if(!response.ok)throw Error('Unable to load the Home Controller warning.');
  const text=response.text();if(!text.trim())throw Error('The Home Controller warning is empty.');return text;
 });},
 gate:function(text,render){
  if(accepted(text))return render();
  const host=E('div',{class:'cbi-map hc-warning'},[E('h2',{},'Home Controller: disclaimer')]);
  const error=E('p',{role:'alert'});
  const button=E('button',{type:'button',class:'cbi-button cbi-button-action',click:async function(){
   button.disabled=true;error.textContent='';acknowledgedText=text;
   try{window.localStorage.setItem(key,text);}catch(e){}
   try{host.replaceChildren(await render());}catch(e){button.disabled=false;error.textContent='Unable to load this page: '+e.message;}
  }},'I understand — continue');
  host.append(E('div',{class:'cbi-section'},[fullText(text),
   E('p',{},'Acknowledgement is remembered in this browser for this router. It does not change the running service or scheduler.'),button,error]));
  return host;
 },
 show:function(text){ui.showModal('Home Controller: disclaimer',[fullText(text),E('div',{class:'right'},[
  E('button',{type:'button',class:'cbi-button cbi-button-neutral',click:()=>ui.hideModal()},'Close')
 ])]);}
});
