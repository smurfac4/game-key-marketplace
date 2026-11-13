#!/usr/bin/env python3
"""Original project-native SVG illustrations. No external artwork or franchise marks."""
from pathlib import Path
import json,random,html,textwrap
root=Path(__file__).resolve().parents[1]
games=json.loads((root/'api/internal/seed/showcase.json').read_text())
def artwork(g,banner=False):
 rng=random.Random(g['app_id']);hue=g['app_id']%280
 sky=''.join(f'<circle cx="{rng.randrange(600)}" cy="{rng.randrange(650)}" r="{rng.choice([1,1,2])}" fill="white" opacity="{rng.uniform(.2,.8):.2f}"/>' for _ in range(100))
 scenery=''
 genre=g['genre']
 if genre in ['RPG','Strategy']:
  scenery='<circle cx="430" cy="235" r="125" fill="url(#moon)"/><path d="M-30 680 140 380 255 540 390 340 640 680Z" fill="#172849"/>'
  for x,y,w in [(130,420,70),(230,310,130),(390,400,65)]:
   scenery+=f'<path d="M{x} 720V{y}l{w/2}-{w/2} {w/2} {w/2}V720Z" fill="#17102e" stroke="#bd99e8" stroke-width="2"/>'
   for dy in range(y+40,680,45):scenery+=f'<rect x="{x+w/2-6}" y="{dy}" width="12" height="22" rx="6" fill="#f6cf7c"/>'
  scenery+='<path d="M280 900Q200 720 290 650L320 650Q300 750 420 900" fill="#9580ba" opacity=".5"/>'
 elif genre=='Racing':
  scenery='<path d="M-100 900 260 340H340L710 900" fill="#151524"/><path d="M300 410 300 900" stroke="#ffd777" stroke-width="9" stroke-dasharray="50 40"/>'
  scenery+='<g transform="translate(90 430)"><ellipse cx="210" cy="200" rx="240" ry="65" fill="#ec4899" opacity=".25"/><path d="M15 170 45 90 120 45H285l70 50 35 75Z" fill="#63e4e3" stroke="#d0ffff" stroke-width="3"/><path d="m110 95 35-35h125l45 40Z" fill="#1c2858"/><path d="M30 165h350v50H30Z" fill="#23215a"/><path d="M50 172h85m150 0h65" stroke="#fff" stroke-width="12"/><circle cx="85" cy="214" r="29" fill="#0b1029"/><circle cx="325" cy="214" r="29" fill="#0b1029"/></g>'
 elif genre=='Action':
  scenery='<circle cx="460" cy="220" r="95" fill="url(#moon)"/>'
  for x in range(-10,600,45):
   y=rng.randint(280,540);scenery+=f'<rect x="{x}" y="{y}" width="40" height="{760-y}" fill="#130e2e" stroke="#7665ad"/>'
   for yy in range(y+20,720,32):scenery+=f'<path d="M{x+9} {yy}h18" stroke="{rng.choice(["#66e3f2","#fc72cf","#e0ba71"])}" stroke-width="4"/>'
  scenery+='<path d="M0 760 300 630 600 760v140H0Z" fill="#17132f"/><path d="M300 640 280 900M300 640 480 900" stroke="#e969d5" stroke-width="3"/>'
 elif genre=='Simulation':
  scenery='<circle cx="420" cy="210" r="100" fill="url(#moon)"/><path d="M0 650Q200 350 600 590V900H0Z" fill="#28585e"/><path d="M0 750Q300 540 600 700V900H0Z" fill="#203e4e"/>'
  for x,y in [(85,560),(260,470),(430,610)]:scenery+=f'<g transform="translate({x} {y})"><path d="m0 40 70-50 70 50v100H0Z" fill="#aa799c"/><path d="m-12 40 82-62 82 62" fill="#efad86"/><rect x="60" y="80" width="25" height="60" fill="#37284e"/><rect x="15" y="65" width="25" height="25" fill="#fce4a8"/></g>'
 else:
  scenery='<circle cx="300" cy="345" r="155" fill="none" stroke="#c9a4fd" stroke-width="8"/><circle cx="300" cy="345" r="130" fill="url(#moon)"/>'
  for x,y in [(-50,200),(70,370),(470,240),(570,380)]:scenery+=f'<path d="M{x} 820V{y}m-80 140 80-130 80 130m-110 90 30-100 70 100" stroke="#172b40" stroke-width="25" fill="none"/>'
  scenery+='<path d="M0 900 260 560h80L600 900" fill="#182b40"/><path d="M280 600 240 900m80-300 80 300" stroke="#54e4d0" opacity=".5" stroke-width="3"/>'
 title=''.join(f'<text x="35" y="{770+i*40}" font-family="sans-serif" font-size="30" font-weight="700" fill="white">{html.escape(t)}</text>' for i,t in enumerate(textwrap.wrap(g['name'],29)[:3]))
 vb='0 130 600 600' if banner else '0 0 600 900';size='width="1600" height="550"' if banner else 'width="600" height="900"'
 return f'<svg xmlns="http://www.w3.org/2000/svg" {size} viewBox="{vb}"><defs><linearGradient id="sky" x2="0" y2="1"><stop stop-color="hsl({hue},65%,20%)"/><stop offset="1" stop-color="#0d122b"/></linearGradient><radialGradient id="moon"><stop stop-color="#f4c58a" stop-opacity=".8"/><stop offset="1" stop-color="#ab75db" stop-opacity=".05"/></radialGradient><linearGradient id="shade" x2="0" y2="1"><stop stop-color="#0d122b" stop-opacity="0"/><stop offset="1" stop-color="#0d122b"/></linearGradient></defs><path d="M0 0h600v900H0Z" fill="url(#sky)"/>{sky}{scenery}<path d="M0 700h600v200H0Z" fill="url(#shade)"/>{"" if banner else title}<text x="35" y="50" font-family="sans-serif" font-size="12" letter-spacing="4" fill="#cfb9ee">KEYSPACE</text></svg>'
for g in games:(root/f'web/public/media/games/{g["app_id"]}.svg').write_text(artwork(g))
for app in [1091500,1245620,1174180]:
 g=next(x for x in games if x['app_id']==app);(root/f'web/public/media/banners/{app}.svg').write_text(artwork(g,True))
print('Generated 50 original cover illustrations and 3 banners.')
