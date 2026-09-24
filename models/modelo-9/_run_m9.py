import json
from pathlib import Path

nb = json.loads(Path(__file__).with_name("modelo_9.ipynb").read_text(encoding="utf-8"))
src = "\n".join("".join(c["source"]) for c in nb["cells"] if c["cell_type"] == "code")
exec(compile(src, "modelo_9.ipynb", "exec"), {"__name__": "__main__"})
