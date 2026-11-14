from pathlib import Path
from io import BytesIO
import json
from PIL import Image

root = Path(__file__).resolve().parents[1] / "docs" / "screenshots"
manifest = {}
for name in ["homepage", "store-filters", "typo-search", "product-sellers", "checkout", "keys-library"]:
    path = root / f"{name}.png"
    original = path.read_bytes()
    with Image.open(BytesIO(original)) as image:
        rgba = image.convert("RGBA")
        assert image.width == 1280
        mode = "RGB" if rgba.getchannel("A").getextrema() == (255, 255) else "RGBA"
        optimized = BytesIO()
        rgba.convert(mode).save(optimized, format="PNG", optimize=True, compress_level=9)
        candidate = optimized.getvalue()
        with Image.open(BytesIO(candidate)) as check:
            assert check.convert("RGBA").tobytes() == rgba.tobytes()
        if len(candidate) < len(original):
            path.write_bytes(candidate)
        final = path.read_bytes()
        manifest[name] = {"width": image.width, "height": image.height, "original_bytes": len(original), "optimized_bytes": len(final)}
        print(f"{name}: {len(original)} -> {len(final)} bytes; pixels preserved")
(root / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
