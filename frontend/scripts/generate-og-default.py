# Generator public/og-default.png (1200x630, palet DESIGN.md §1).
# Cara pakai: unduh dulu Space Grotesk variable font, lalu:
#   python frontend/scripts/generate-og-default.py <path-ke-ttf>
# Contoh font: https://github.com/google/fonts/raw/main/ofl/spacegrotesk/SpaceGrotesk%5Bwght%5D.ttf
# (variable font di-set ke wght 700 = Bold sesuai tipografi display).
import sys

from PIL import Image, ImageDraw, ImageFont

W, H = 1200, 630
BG, INK = "#FAFAF7", "#1C1A12"

img = Image.new("RGB", (W, H), BG)
d = ImageDraw.Draw(img)

bw = 24
d.rectangle([bw // 2, bw //2, W - bw // 2 - 1, H - bw // 2 - 1], outline=INK, width=bw)

size = 230
while size > 40:
    font = ImageFont.truetype(sys.argv[1], size)
    try:
        font.set_variation_by_axes([700])
    except Exception:
        pass
    left, _, right, _ = d.textbbox((0, 0), "JEJAK", font=font)
    if right - left <= 900:
        break
    size -= 10

d.text((W / 2, H / 2), "JEJAK", font=font, fill=INK, anchor="mm")
img.save("frontend/public/og-default.png")
print(f"saved frontend/public/og-default.png ({W}x{H}, font size {size})")
