#!/usr/bin/env bash
# 写扩散压测：模拟大 V 上传视频，测 ①上传接口延迟 ②后台扩散完成时间
# 对比两版：退化版同步 INSERT feed 表（上传被卡住）vs 优化版 MQ 异步 + Redis（上传秒回）
#
# 用法: ./fanout_test.sh <base_url> [rounds]
#   例: ./fanout_test.sh http://localhost:8081 3   # 退化版
#       ./fanout_test.sh http://localhost:2222 3   # 优化版
# 前置: 已 seed -bigv-fans（产出 ../bigv.txt，含大V + 前3粉丝 token）
# 扩散完成判定: 轮询前 3 个粉丝的 /feed 接口，全部含刚上传的视频 = 扩散完成
set -euo pipefail

EXP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BASE="$1"
ROUNDS="${2:-3}"
BIGV_FILE="$EXP_DIR/bigv.txt"
FAN_SAMPLE=3          # 轮询的粉丝样本数（bigv.txt 里的 fan: 行）
TIMEOUT_S=300         # 扩散超时保护

# 版本判定（约定端口：2222=优化版 8081=退化版）
case "$BASE" in
  *:2222*) VERSION="优化版" ;;
  *:8081*) VERSION="退化版" ;;
  *)       VERSION="未知(按端口判 2222/8081)" ;;
esac

[ -f "$BIGV_FILE" ] || { echo "缺少 $BIGV_FILE，请先运行 seed -bigv-fans"; exit 1; }
BIGV_TOKEN=$(grep '^bigv:' "$BIGV_FILE" | awk '{print $2}')
FAN_LINES=$(grep '^fan:' "$BIGV_FILE" | head -n "$FAN_SAMPLE")
[ -n "$BIGV_TOKEN" ] && [ -n "$FAN_LINES" ] || { echo "bigv.txt 格式错误"; exit 1; }

# 造一个最小 3s 视频（保证 ffmpeg 能抽帧做封面）
ffmpeg -y -f lavfi -i color=c=black:s=320x240:d=3 -c:v libx264 -preset ultrafast -f mp4 /tmp/fanout_v.mp4 >/dev/null 2>&1

# 检查粉丝 feed 是否已含视频 vid
feed_has() {
  local url=$1 token=$2 vid=$3
  curl -s "$url/feed?offset=0&count=50" -H "Authorization: Bearer $token" \
    | python3 -c "import sys,json
d=json.load(sys.stdin)
ids=[v['id'] for v in d.get('data',{}).get('videos',[])]
sys.exit(0 if $vid in ids else 1)"
}

echo "===== 写扩散压测: $BASE（$VERSION）====="
echo "大V 每发一条视频 → fanout 推给全部粉丝（$FAN_SAMPLE 个粉丝样本检测扩散完成）"
echo ""

UPLOAD_SUM=0
SPREAD_SUM=0
for r in $(seq 1 "$ROUNDS"); do
  # 1) 大 V 上传视频，测上传接口延迟
  T0=$(date +%s%N)
  RESP=$(curl -s -X POST "$BASE/video/upload" -H "Authorization: Bearer $BIGV_TOKEN" \
    -F "file=@/tmp/fanout_v.mp4;filename=fanout.mp4" -F "title=大V扩散压测" -F "describe=fanout")
  T1=$(date +%s%N)
  VID=$(echo "$RESP" | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["videoId"])')
  UPLOAD_MS=$(( (T1 - T0) / 1000000 ))

  # 2) 轮询粉丝 feed，测扩散完成时间
  START=$(date +%s%N)
  done_flag=0
  while [ "$done_flag" -eq 0 ]; do
    done_flag=1
    while IFS= read -r line; do
      FTOKEN=$(echo "$line" | awk '{print $2}')
      if ! feed_has "$BASE" "$FTOKEN" "$VID"; then
        done_flag=0
      fi
    done <<< "$FAN_LINES"

    if [ "$done_flag" -eq 0 ]; then
      NOW=$(date +%s%N)
      ELAPSED_S=$(( (NOW - START) / 1000000000 ))
      if [ "$ELAPSED_S" -gt "$TIMEOUT_S" ]; then
        echo "  第 $r 次扩散超时（>${TIMEOUT_S}s），中断本轮"
        done_flag=1
        break
      fi
      sleep 1
    fi
  done
  END=$(date +%s%N)
  SPREAD_MS=$(( (END - START) / 1000000 ))

  UPLOAD_SUM=$((UPLOAD_SUM + UPLOAD_MS))
  SPREAD_SUM=$((SPREAD_SUM + SPREAD_MS))
  echo "  round $r: 上传接口 ${UPLOAD_MS}ms · 扩散完成 ${SPREAD_MS}ms（videoId=$VID）"
done

echo ""
echo "  平均: 上传接口 $((UPLOAD_SUM / ROUNDS))ms · 扩散完成 $((SPREAD_SUM / ROUNDS))ms"
echo ""
echo "解读：退化版『上传接口』≈ 同步写 feed 表耗时；优化版『上传接口』≈ MQ 异步丢消息耗时（毫秒级），"
echo "      『扩散完成』才是后台实际推完全部粉丝的时间。两版差距 = 异步化的价值。"
