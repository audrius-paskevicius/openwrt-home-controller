// Minimal widget/RPC fixture for isolated editor tests. Native LuCI is also
// checked on the target router; these doubles do not test the LuCI framework.
window.ui={
 Textfield:class {
  constructor(value,options){this.value=value;this.options=options;}
  render(){return E('div',{},E('input',{class:'cbi-input-text',type:this.options.password?'password':'text',value:this.value}));}
 },
 Select:class {
  constructor(value,choices){this.value=value;this.choices=choices;}
  render(){const input=E('select',{class:'cbi-input-select'},Object.entries(this.choices).map(([value,label])=>E('option',{value},label)));input.value=this.value;return E('div',{},input);}
 },
 Checkbox:class {
  constructor(value){this.value=value;}
  render(){const input=E('input',{type:'checkbox',id:'test-'+Math.random()});input.checked=this.value==='1';return E('div',{class:'cbi-checkbox'},[input,E('label',{for:input.id})]);}
 },
 tabs:{initTabGroup(panes){const menu=E('ul',{class:'cbi-tabmenu'}),show=chosen=>panes.forEach(p=>{p.hidden=p!==chosen;p.setAttribute('data-tab-active',String(p===chosen));});for(const p of panes)menu.append(E('li',{class:'cbi-tab'},E('a',{href:'#',click:e=>{e.preventDefault();show(p);}},p.getAttribute('data-tab-title'))));panes[0].parentNode.before(menu);show(panes[0]);}},
 showModal:(title,children)=>{ui.hideModal();document.body.append(E('div',{id:'modal'},[E('h2',{},title),...children]));},
 hideModal:()=>document.getElementById('modal')?.remove()
};
