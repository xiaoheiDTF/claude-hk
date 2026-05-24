"""JSON field extractor for hook scripts.

Usage: printf '<json>' | python json_get.py <key>
  e.g.: printf '<json>' | python json_get.py .hook_event_name
        printf '<json>' | python json_get.py .tool_name
"""
import sys
import json

data = json.load(sys.stdin)
keys = sys.argv[1].lstrip(".").split(".")
v = data
for k in keys:
    if isinstance(v, dict) and k in v:
        v = v[k]
    else:
        v = ""
        break
print(v if not isinstance(v, (list, dict)) else json.dumps(v, ensure_ascii=False))
