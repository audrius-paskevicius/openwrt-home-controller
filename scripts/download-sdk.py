"""Bounded, resumable range downloads; final digest always verified by caller."""
import concurrent.futures, pathlib, sys, time, urllib.request
url, destination = sys.argv[1:]
out = pathlib.Path(destination)
with urllib.request.urlopen(urllib.request.Request(url, method='HEAD'), timeout=30) as r:
    size = int(r.headers['Content-Length'])
if out.exists() and out.stat().st_size == size:
    sys.exit(0)
parts = out.parent / 'sdk-parts'
parts.mkdir(exist_ok=True)
chunk = 4 * 1024 * 1024
def fetch(i):
    start, end = i*chunk, min((i+1)*chunk,size)-1
    path = parts / str(i)
    if path.exists() and path.stat().st_size == end-start+1:
        return
    for attempt in range(5):
        try:
            req = urllib.request.Request(url, headers={'Range':'bytes=%d-%d'%(start,end)})
            with urllib.request.urlopen(req,timeout=45) as r:
                if r.status != 206 or not r.headers.get('Content-Range','').startswith('bytes %d-'%start):
                    raise RuntimeError('Server did not honor range')
                data = r.read(chunk+1)
            if len(data) != end-start+1:
                raise RuntimeError('Incomplete range')
            path.write_bytes(data)
            print('SDK segment %d complete' % i, flush=True)
            return
        except Exception:
            if attempt == 4: raise
            time.sleep(2*(attempt+1))
count=(size+chunk-1)//chunk
with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
    list(pool.map(fetch,range(count)))
with out.open('wb') as f:
    for i in range(count):
        f.write((parts/str(i)).read_bytes())
