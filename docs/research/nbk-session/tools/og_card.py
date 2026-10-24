from PIL import Image, ImageDraw, ImageFont
F = '/tmp/claude-0/-home-user-darkpawns/950883f4-8a1e-543f-8181-0ecc0f69af13/scratchpad/fonts/'
PAPER, INK, MUTED, OX = (239,231,214), (26,22,20), (86,80,74), (168,32,26)
CRT, FG, GRN, RED = (10,9,8), (200,184,150), (13,188,121), (205,49,49)
im = Image.new('RGB', (1200, 630), PAPER); d = ImageDraw.Draw(im)
d.rectangle((40,40,1159,589), outline=INK, width=3)
d.rectangle((48,48,1151,581), outline=INK, width=1)
serif = lambda s: ImageFont.truetype(F+'dmserif.ttf', s)
d.text((100,155), 'DARK PAWNS  ·  THE BLOG', font=serif(24), fill=OX)
d.text((100,205), 'And You Wanna', font=serif(76), fill=INK)
d.text((100,290), 'Know Why?', font=serif(76), fill=INK)
d.line((100,410,640,410), fill=INK, width=1)
d.text((100,430), 'darkpawns.org', font=serif(28), fill=MUTED)
# the game panel
x0,y0,x1,y1 = 690,170,1110,440
d.rectangle((x0,y0,x1,y1), fill=CRT, outline=INK, width=2)
mono = ImageFont.truetype(F+'jbm.ttf', 17)
lh = 27; x = x0+24; y = y0+28
def runs(parts):
    global y
    cx = x
    for text, col in parts:
        d.text((cx,y), text, font=mono, fill=col); cx += d.textlength(text, font=mono)
    y += lh
prompt = [('262',GRN),('H ',FG),('100',GRN),('M ',FG),('126',GRN),('V >',FG)]
runs(prompt)
runs([("Mickey Knox asks, 'And you",FG)])
runs([("wanna know why?'",FG)])
y += lh/2
runs(prompt)
runs([("Mickey Knox says, 'Shit $N,",FG)])
runs([("I'm a natural born killer.'.",FG)])
runs([("Mickey Knox hits you very hard.",RED)])
im.save('/home/user/darkpawns/website/static/images/blog/og-mickey-and-mallory.png', optimize=True)
