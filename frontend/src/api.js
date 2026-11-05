// Use generated Wails 3 bindings in the desktop app and isolated localStorage
// data in a normal browser preview.
import * as AppBinding from '../bindings/dailyflow/app.js';

const key = 'dailyflow-svelte-preview-v1';
const today = () => {const d=new Date();return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`};
const seed = () => ({
  tasks: [
    {id:'demo1',title:'Implement Laravel API',category:'Work',date:today(),start:'09:00',duration:120,repeat:'weekdays',weekdays:[],notes:'Refactor endpoints and add validation'},
    {id:'demo2',title:'Review React dashboard',category:'Work',date:today(),start:'11:00',duration:60,repeat:'weekdays',weekdays:[],notes:''},
    {id:'demo3',title:'Develop Iploy deployment',category:'Project',date:today(),start:'13:00',duration:120,repeat:'daily',weekdays:[],notes:''},
    {id:'demo4',title:'Study Mandarin — TOCFL A2',category:'Learning',date:today(),start:'19:00',duration:60,repeat:'daily',weekdays:[],notes:''},
    {id:'demo5',title:'Read AI Engineering paper',category:'Learning',date:today(),start:'20:30',duration:30,repeat:'weekly',weekdays:[],notes:''}
  ],
  results: [{taskId:'demo1',date:today(),done:true,activity:'Built API endpoint',output:'Endpoints tested',summary:'Validation complete',actualMinutes:90},{taskId:'demo2',date:today(),done:true,activity:'Reviewed components',output:'2 fixes',summary:'Layout improved',actualMinutes:55}]
});
const data = () => {try {return JSON.parse(localStorage.getItem(key))||seed()}catch{return seed()}};
const write = d => localStorage.setItem(key,JSON.stringify(d));
const occurs = (t,date) => {
  if(date<t.date)return false;
  const dow=new Date(`${date}T12:00:00`).getDay(), start=new Date(`${t.date}T12:00:00`).getDay();
  return t.repeat==='daily'||t.repeat==='weekdays'&&dow!==0&&dow!==6||t.repeat==='weekly'&&dow===start||t.repeat==='custom'&&t.weekdays?.includes(dow)||(!t.repeat||t.repeat==='none')&&date===t.date;
};
const mock={
  async AllTasks(){return data().tasks},
  async AllResults(){return data().results},
  async AllResults(){return data().results},
  async Day(date){const d=data();const entries=d.tasks.filter(t=>occurs(t,date)).map(task=>({task,result:d.results.find(r=>r.taskId===task.id&&r.date===date)||{taskId:task.id,date,done:false,activity:'',output:'',summary:'',actualMinutes:0}})).sort((a,b)=>a.task.start.localeCompare(b.task.start));const done=entries.filter(e=>e.result.done).length;return {date,entries,total:entries.length,done,percent:entries.length?Math.round(done/entries.length*100):0,summary:entries.map(e=>e.result.summary).filter(Boolean).join('\n')}},
  async SaveTask(task){if(!task.title?.trim())throw Error('Title is required');const d=data();if(!task.id)task.id=crypto.randomUUID();const i=d.tasks.findIndex(t=>t.id===task.id);if(i<0)d.tasks.push(task);else d.tasks[i]=task;write(d)},
  async DeleteTask(id){const d=data();d.tasks=d.tasks.filter(t=>t.id!==id);d.results=d.results.filter(r=>r.taskId!==id);write(d)},
  async SaveResult(r){if(r.done&&!r.activity?.trim())throw Error('Activity is required before completing a task');const d=data();const i=d.results.findIndex(x=>x.taskId===r.taskId&&x.date===r.date);if(i<0)d.results.push(r);else d.results[i]=r;write(d)},
  async SaveAIEvaluation(taskID,date,score,evaluation){const d=data();const result=d.results.find(x=>x.taskId===taskID&&x.date===date&&x.done);if(!result)throw Error('Completed activity result was not found');result.aiScore=score;result.aiEvaluation=evaluation;write(d)},
  async GetAISettings(){return {defaultProvider:'',providers:{openai:{connected:false,model:'gpt-6-luna'},gemini:{connected:false,model:'gemini-3.8-flash'},groq:{connected:false,model:'openai/gpt-oss-120b'}}}},
  async ConnectAI(){throw Error('AI settings are available in the DailyFlow desktop app, not browser preview.')},
  async SetAIDefaultProvider(){throw Error('AI settings are available in the DailyFlow desktop app, not browser preview.')},
  async SetAIProviderModel(){throw Error('AI settings are available in the DailyFlow desktop app, not browser preview.')},
  async DisconnectAI(){return {defaultProvider:'',providers:{openai:{connected:false,model:'gpt-6-luna'},gemini:{connected:false,model:'gemini-3.8-flash'},groq:{connected:false,model:'openai/gpt-oss-120b'}}}},
  async EvaluateActivity(){throw Error('AI evaluation is available in the DailyFlow desktop app.')},
  async GetTrayIconEnabled(){return JSON.parse(localStorage.getItem('dailyflow-tray-icon-enabled')??'true')},
  async SetTrayIconEnabled(enabled){localStorage.setItem('dailyflow-tray-icon-enabled',JSON.stringify(enabled))},
  async ToggleOverview(){window.dailyFlowSetMini?.(true)},
  async OpenActivityResult(taskID,date){window.dailyFlowSetMini?.(false);await window.dailyFlowOpenResult?.(taskID,date)},
  async RestoreFull(){window.dailyFlowSetMini?.(false)},
  async ExitApp(){window.alert('Preview only: close the browser tab to exit.')}
};
export const isPreview = !window._wails;
export const api = new Proxy({}, {get:(_,method)=>(...args)=>isPreview?mock[method](...args):AppBinding[method](...args)});
