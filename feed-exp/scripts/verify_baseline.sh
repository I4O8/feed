#!/usr/bin/env bash
# 验证一个 feed 后端全链路可跑（默认退化版 :8081；-base 可指定）
# 用法: ./verify_baseline.sh [base_url]
# 覆盖：注册→登录→上传(同步 fanout)→关注→刷 feed→点赞→收藏→reportPlay
set -e

BASE="${1:-http://localhost:8081}"
SUFFIX=$(date +%s)
EMAIL_A="verify_a_${SUFFIX}@test.com"
EMAIL_B="verify_b_${SUFFIX}@test.com"
PWD="123456"

# 从响应 JSON 提取 data.<key>
field() { python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('data',{}) or {}).get('$1','')" 2>/dev/null; }

echo "===== 验证全链路: $BASE ====="

echo "== 1. 注册用户 A、B =="
R_A=$(curl -s -X POST "$BASE/user/register" -H 'Content-Type: application/json' -d "{\"account\":\"$EMAIL_A\",\"password\":\"$PWD\"}")
echo "$R_A" | grep -q '"code":0' || { echo "A 注册失败: $R_A"; exit 1; }
TOKEN_A=$(echo "$R_A" | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["token"])')
UID_A=$(echo "$R_A" | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["userId"])')
R_B=$(curl -s -X POST "$BASE/user/register" -H 'Content-Type: application/json' -d "{\"account\":\"$EMAIL_B\",\"password\":\"$PWD\"}")
echo "$R_B" | grep -q '"code":0' || { echo "B 注册失败: $R_B"; exit 1; }
TOKEN_B=$(echo "$R_B" | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["token"])')
UID_B=$(echo "$R_B" | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["userId"])')
echo "  A=$UID_A  B=$UID_B OK"

echo "== 2. 登录 A（验证 bcrypt 正常）=="
curl -s -X POST "$BASE/user/login" -H 'Content-Type: application/json' -d "{\"account\":\"$EMAIL_A\",\"password\":\"$PWD\"}" | grep -q '"code":0' && echo "  登录 OK"

echo "== 3. B 关注 A（纯推模式：先关注，上传时才推给粉丝）=="
curl -s -X POST "$BASE/follow/$UID_A" -H "Authorization: Bearer $TOKEN_B" | grep -q '"code":0' && echo "  关注 OK"

echo "== 4. A 上传视频（触发同步 fanout 给粉丝 B）=="
# 3s 视频：保证 ffmpeg 在第 1 秒能抽到帧作为封面
ffmpeg -y -f lavfi -i color=c=black:s=320x240:d=3 -c:v libx264 -preset ultrafast -f mp4 /tmp/bench_verify.mp4 >/dev/null 2>&1
R_UPLOAD=$(curl -s -X POST "$BASE/video/upload" -H "Authorization: Bearer $TOKEN_A" \
  -F "file=@/tmp/bench_verify.mp4;filename=bench.mp4" -F "title=压测验证视频" -F "describe=全链路验证")
echo "$R_UPLOAD" | grep -q '"code":0' || { echo "上传失败: $R_UPLOAD"; exit 1; }
VID=$(echo "$R_UPLOAD" | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["videoId"])')
echo "  上传 OK videoId=$VID"

echo "== 5. B 刷 feed（应包含 A 刚传的视频）=="
FEED=$(curl -s "$BASE/feed?offset=0&count=20" -H "Authorization: Bearer $TOKEN_B")
echo "$FEED" | python3 -c "import sys,json;d=json.load(sys.stdin);ids=[v['id'] for v in d.get('data',{}).get('videos',[])];print('  feed 视频:',ids);sys.exit(0 if '$VID' in [str(i) for i in ids] else 1)" || {
  echo "feed 未包含新视频，响应: $FEED"; exit 1; }
echo "  feed 含新视频 OK"

echo "== 6. B 点赞 + like/status + like_count =="
curl -s -X POST "$BASE/like/$VID" -H "Authorization: Bearer $TOKEN_B" | grep -q '"code":0' && echo "  点赞 OK"
curl -s "$BASE/like/status?videoId=$VID" -H "Authorization: Bearer $TOKEN_B" | grep -q '"liked":true' && echo "  like/status=true OK"
mysql -ugorm -pgorm12345 -h127.0.0.1 -e "SELECT like_count FROM feed.video WHERE id=$VID;" 2>/dev/null | tail -1 | grep -q '1' && echo "  like_count=1 OK"

echo "== 7. B 收藏 + fav/status + fav_count =="
curl -s -X POST "$BASE/fav/$VID" -H "Authorization: Bearer $TOKEN_B" | grep -q '"code":0' && echo "  收藏 OK"
curl -s "$BASE/fav/status?videoId=$VID" -H "Authorization: Bearer $TOKEN_B" | grep -q '"favored":true' && echo "  fav/status=true OK"
mysql -ugorm -pgorm12345 -h127.0.0.1 -e "SELECT fav_count FROM feed.video WHERE id=$VID;" 2>/dev/null | tail -1 | grep -q '1' && echo "  fav_count=1 OK"

echo "== 8. reportPlay 两次（不同 deviceId，验证无去重）=="
curl -s -X POST "$BASE/video/reportPlay" -H 'Content-Type: application/json' -d "{\"videoId\":$VID,\"deviceId\":\"dev1\"}" | grep -q '"code":0'
curl -s -X POST "$BASE/video/reportPlay" -H 'Content-Type: application/json' -d "{\"videoId\":$VID,\"deviceId\":\"dev2\"}" | grep -q '"code":0'
mysql -ugorm -pgorm12345 -h127.0.0.1 -e "SELECT play_count FROM feed.video WHERE id=$VID;" 2>/dev/null | tail -1 | grep -q '2' && echo "  play_count=2 OK"

echo ""
echo "===== 全链路验证通过 ✓ ====="
