#!/usr/bin/env sh

PATH="/home/amirmohammad/Git/rate-limit-checker/claude/visualizer/"
OUT="$PATH/report.html"

echo '<!DOCTYPE html><html><head><style>' > "$OUT"
echo 'body{background:#111;color:#ccc;font-family:monospace;padding:20px;display:flex;flex-wrap:wrap;gap:16px;}' >> "$OUT"
echo '.file{border:1px solid #333;padding:8px;}' >> "$OUT"
echo 'h2{font-size:11px;color:#666;margin:0 0 6px;}' >> "$OUT"
echo '.grid{display:flex;flex-wrap:wrap;gap:2px;width:110px;}' >> "$OUT"
echo '.s{width:10px;height:10px;}' >> "$OUT"
echo '.ok{background:#4a4;}.limited{background:#a44;}.timeout{background:#555;}' >> "$OUT"
echo '</style></head><body>' >> "$OUT"

for f in "$@"; do
  echo "<div class=\"file\"><h2>$(basename "$f")</h2><div class=\"grid\">" >> "$OUT"
  grep -o '\] worker=[0-9]*  [0-9]*' "$f" | grep -o '[0-9]*$' | while read -r code; do
    case "$code" in
      200) echo '<div class="s ok"></div>' >> "$OUT" ;;
      429) echo '<div class="s limited"></div>' >> "$OUT" ;;
      *)   echo '<div class="s timeout"></div>' >> "$OUT" ;;
    esac
  done
  echo "</div></div>" >> "$OUT"
done

echo '</body></html>' >> "$OUT"
echo "done → $OUT"
