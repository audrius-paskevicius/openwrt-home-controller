/* SPDX-License-Identifier: GPL-3.0-or-later */
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
int main(int argc,char **argv) {
 if(argc!=3) return 2;
 char *end; long id=strtol(argv[1],&end,10);
 if(*end || id<1 || id>65535) return 2;
 int mask; const char *method;
 if(!strcmp(argv[2],"up")){mask=128;method="tdUp";}
 else if(!strcmp(argv[2],"down")){mask=256;method="tdDown";}
 else if(!strcmp(argv[2],"stop")){mask=512;method="tdStop";}
 else if(!strcmp(argv[2],"on")){mask=1;method="tdTurnOn";}
 else if(!strcmp(argv[2],"off")){mask=2;method="tdTurnOff";}
 else return 2;
 alarm(8);
 void *lib=dlopen("libtelldus-core.so.2",RTLD_NOW);
 if(!lib){fprintf(stderr,"Telldus library unavailable\n");return 3;}
 void (*init)(void)=dlsym(lib,"tdInit"),(*closefn)(void)=dlsym(lib,"tdClose");
 int (*methods)(int,int)=dlsym(lib,"tdMethods"),(*call)(int)=dlsym(lib,method);
 if(!init||!closefn||!methods||!call) return 3;
 init(); int result=(methods(id,mask)&mask)?call(id):-998;
 closefn(); printf("%d\n",result); dlclose(lib);
 return result?1:0;
}
