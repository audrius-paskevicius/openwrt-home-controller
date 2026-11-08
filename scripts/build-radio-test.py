import pathlib, shlex, subprocess, sys
linkfile=pathlib.Path(sys.argv[1]).resolve()
root=pathlib.Path(sys.argv[2]).resolve()
service=linkfile.parents[2]
flags={}
for line in (linkfile.parent/'flags.make').read_text().splitlines():
    if ' = ' in line:
        key,value=line.split(' = ',1);flags[key]=shlex.split(value)
link=shlex.split(linkfile.read_text())
obj=root/'dist/hasta-test.o'
subprocess.run([link[0]]+flags['CXX_DEFINES']+flags['CXX_INCLUDES']+flags['CXX_FLAGS']+['-c',str(root/'tests/hasta.cpp'),'-o',str(obj)],cwd=service,check=True)
main=[a for a in link if a.endswith('/main_unix.cpp.o')]
assert len(main)==1,main
link.remove(main[0]);link.insert(1,str(obj));link[link.index('-o')+1]=str(root/'dist/hasta-test')
for directory in sorted({str(pathlib.Path(a).parent) for a in link if a.startswith('/') and a.endswith('.so')}):
    link.append('-Wl,-rpath-link,'+directory)
subprocess.run(link,cwd=service,check=True)
obj.unlink()
