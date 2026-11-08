const fs=require('node:fs');
for(const file of ['web/app.js','luci/home-controller.js']){
 new Function(fs.readFileSync(file,'utf8'));
 console.log('Syntax OK:',file);
}
