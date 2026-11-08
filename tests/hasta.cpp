// SPDX-License-Identifier: GPL-2.0-or-later
// Synthetic addresses only. This program calculates packets; it never sends RF.
#include "service/ProtocolHasta.h"
#include <cstdlib>
#include <iostream>
#include <vector>
static int checks=0;
static void check(bool ok,const char *name){++checks;if(!ok){std::cerr<<"FAIL: "<<name<<std::endl;std::exit(1);}}
static std::string decode(const char *model,const char *data){return ProtocolHasta::decodeData(ControllerMessage(std::string("protocol:hasta;model:")+model+";data:"+data+";"));}
static std::string wave(const wchar_t *model,const wchar_t *unit,int method){ProtocolHasta p;ParameterMap params;params[L"house"]=L"12345";if(unit)params[L"unit"]=unit;p.setModel(model);p.setParameters(params);return p.getStringForMethod(method,0,NULL);}
static std::vector<unsigned> payload(const std::string &w){
 check(w.size()==92,"v2 pulse count");const unsigned char pre[]={245,1,245,245,63,1,63,1,35,35};
 for(int i=0;i<10;++i)check((unsigned char)w[i]==pre[i],"v2 preamble");
 std::vector<unsigned>b(5,0);
 for(int j=0;j<5;++j)for(int k=0;k<8;++k){unsigned a=(unsigned char)w[10+j*16+k*2],z=(unsigned char)w[11+j*16+k*2];check((a==63&&z==35)||(a==35&&z==63),"valid pulse pair");if(a==63)b[j]|=1U<<k;}
 check((unsigned char)w[90]==63&&(unsigned char)w[91]==35,"trailer");check(((b[0]+b[1]+b[2]+b[3]+b[4])&255)==1,"checksum");return b;
}
int main(){
 const char *frames[]={"0x3039C0C0","0x303950C0","0x303910C0"};const char *methods[]={"up","stop","down"};
 const int codes[]={TELLSTICK_UP,TELLSTICK_STOP,TELLSTICK_DOWN};const unsigned bytes[]={0xC0,0x50,0x10};
 for(int i=0;i<3;++i){
  std::string expected=std::string("class:command;protocol:hasta;model:selflearningv2;house:12345;unit:0;method:")+methods[i]+";";
  check(decode("selflearningv2",frames[i])==expected,"synthetic v2 unit-zero receive");
  std::vector<unsigned>b=payload(wave(L"selflearningv2",L"0",codes[i]));check(b[0]==0x30&&b[1]==0x39&&b[2]==bytes[i]&&b[3]==1,"unit-zero transmit payload");
  b=payload(wave(L"selflearningv2",L"1",codes[i]));check(b[2]==(bytes[i]|1),"unit one preserved");
  check(wave(L"selflearningv2",NULL,codes[i])==wave(L"selflearningv2",L"1",codes[i]),"default unit one");
  b=payload(wave(L"selflearningv2",L"15",codes[i]));check(b[2]==(bytes[i]|15),"unit fifteen preserved");
  check(wave(L"selflearning",L"0",codes[i])==wave(L"selflearning",L"1",codes[i]),"v1 clamping preserved");
 }
 check(decode("selflearning","0xC671100")=="class:command;protocol:hasta;model:selflearning;house:26380;unit:1;method:down;","v1 down fixture");
 check(decode("selflearning","0xC670100")=="class:command;protocol:hasta;model:selflearning;house:26380;unit:1;method:up;","v1 up fixture");
 check(decode("selflearningv2","0x4B891F01")=="class:command;protocol:hasta;model:selflearningv2;house:19337;unit:15;method:down;","v2 down fixture");
 check(decode("selflearningv2","0x4B89CF01")=="class:command;protocol:hasta;model:selflearningv2;house:19337;unit:15;method:up;","v2 up fixture");
 check(decode("selflearning","0xC670000").empty(),"v1 unit zero rejected");
 check(decode("selflearningv2","0x0000C0C0").empty(),"zero house rejected");
 check(decode("selflearningv2","0x3039F0C0").empty(),"unknown method rejected");
 std::cout<<"PASS: "<<checks<<" protocol checks; no RF sent."<<std::endl;
}
