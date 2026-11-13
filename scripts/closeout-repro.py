from pathlib import Path
import tempfile,subprocess,os,secrets,shutil,json,urllib.request,time
root=Path(__file__).resolve().parents[1]; target=Path(tempfile.mkdtemp(prefix='eneba-clean-',dir='/private/tmp'))
files=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','-z'],cwd=root).decode().split('\0')
for f in files:
 if f and (root/f).is_file():
  out=target/f;out.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(root/f,out)
env=os.environ.copy();env.update(POSTGRES_USER='marketplace',POSTGRES_PASSWORD='clean-repro-only',POSTGRES_DB='marketplace',POSTGRES_PORT='15433',API_PORT='18082',WEB_PORT='15175',JWT_SECRET=secrets.token_hex(32),COOKIE_SECURE='false',COOKIE_SAMESITE='lax',AUTH_ALLOWED_ORIGINS='http://localhost:15175')
base=['docker','compose','-p','eneba-closeout-clean']
def run(args,**kwargs):return subprocess.run(args,cwd=target,env=env,check=True,**kwargs)
try:
 print('Filtered fresh tree created; no .env or node_modules copied.',flush=True)
 run(base+['up','--build','-d'])
 run(base+['run','--rm','--no-deps','api','migrate','up'])
 run(base+['run','--rm','--no-deps','api','seed'])
 for endpoint in ['/api/health/ready','/api/products','/api/products/demo-game-001']:
  with urllib.request.urlopen('http://localhost:15175'+endpoint,timeout=10) as r:
   body=json.load(r)
   assert r.status==200
   if endpoint.endswith('demo-game-001'): assert body['data']['image_url'].endswith('.svg')
   print('Smoke HTTP 200:',endpoint,flush=True)
 web=subprocess.check_output(base+['ps','-q','web'],cwd=target,env=env).decode().strip()
 code="""const{chromium}=require('/app/node_modules/playwright');(async()=>{const b=await chromium.launch();const p=await b.newPage();await p.goto('http://localhost:5173/store');await p.locator('a[href^=\"/products/\"]').first().waitFor();await p.goto('http://localhost:5173/products/demo-game-001');await p.getByRole('heading',{name:'Compare sellers'}).waitFor();await b.close();console.log('Clean-copy real browser catalog/details: PASS')})().catch(()=>process.exit(1))"""
 run(['docker','run','--rm','--ipc=host','--network','container:'+web,'-v',str(root)+':/app','mcr.microsoft.com/playwright:v1.58.2-noble','node','-e',code])
finally:
 subprocess.run(base+['down','-v','--remove-orphans'],cwd=target,env=env,check=False)
 shutil.rmtree(target)
 print('Clean environment and source copy removed.',flush=True)
