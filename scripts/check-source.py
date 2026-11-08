"""Check tracked public files only. Deployment directories are never scanned or uploaded."""
import pathlib, re, subprocess, sys
root=pathlib.Path(__file__).resolve().parents[1]
files=subprocess.check_output(['git','ls-files','-z'],cwd=root).decode().split('\0')
bad=[]
for name in filter(None,files):
    if name.startswith(('.private/','.tools/','build/','dist/')):
        bad.append(name+': private/build material tracked')
        continue
    p=root/name
    if not p.is_file(): continue
    data=p.read_bytes()
    if re.search(rb'-----BEGIN [A-Z ]*PRIVATE KEY-----',data):
        bad.append(name+': private key marker')
    if re.search(rb'(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,})',data):
        bad.append(name+': access token')
    if b'\r\n' in data and p.suffix in ('.sh','.go','.py'):
        bad.append(name+': non-portable line endings')
if bad:
    sys.exit('\n'.join(bad))
print('Public source checks passed')
