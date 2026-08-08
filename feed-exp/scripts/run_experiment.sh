#!/usr/bin/env bash
# 一键跑完整性能对比实验（任务书 docs/性能对比实验.md §4.2）
# 每组压 ROUNDS 次，组间自动 seed -reset 清数据污染，结果保留在 results/
#
# 环境变量可覆盖默认值：
#   USERS VIDEOS CONC DUR EXTREME_CONC EXTREME_DUR ROUNDS NOTES
#
# 前置：MySQL/Redis/RabbitMQ 运行中；SET GLOBAL max_connections=1000; ulimit -n 65535
set -euo pipefail

EXP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$EXP_DIR"

USERS=${USERS:-10000}
VIDEOS=${VIDEOS:-50000}
CONC=${CONC:-500}
DUR=${DUR:-60s}
EXTREME_CONC=${EXTREME_CONC:-1500}
EXTREME_DUR=${EXTREME_DUR:-300s}
ROUNDS=${ROUNDS:-3}
BIGV_FANS=${BIGV_FANS:-0}   # >0 时执行写扩散压测（如 200000）
NOTES=${NOTES:-"同一台机器；MySQL/Redis/MQ 本机；连接池 MaxOpenConns=200"}

OPT_URL=${OPT_URL:-http://localhost:2222}
BASE_URL=${BASE_URL:-http://localhost:8081}

# 压测期间临时禁用系统自动熄屏/挂起（GNOME），结束后自动恢复
SAVE_IDLE=""; SAVE_SLEEP_AC=""; SAVE_SLEEP_BAT=""
if command -v gsettings >/dev/null 2>&1; then
  SAVE_IDLE=$(gsettings get org.gnome.desktop.session idle-delay 2>/dev/null || true)
  SAVE_SLEEP_AC=$(gsettings get org.gnome.settings-daemon.plugins.power sleep-inactive-ac-type 2>/dev/null || true)
  SAVE_SLEEP_BAT=$(gsettings get org.gnome.settings-daemon.plugins.power sleep-inactive-battery-type 2>/dev/null || true)
  gsettings set org.gnome.desktop.session idle-delay 0 2>/dev/null || true
  gsettings set org.gnome.settings-daemon.plugins.power sleep-inactive-ac-type 'nothing' 2>/dev/null || true
  gsettings set org.gnome.settings-daemon.plugins.power sleep-inactive-battery-type 'nothing' 2>/dev/null || true
  echo "已临时禁用自动熄屏/挂起（实验结束自动恢复）"
fi
restore_screen() {
  [ -n "$SAVE_IDLE" ] && gsettings set org.gnome.desktop.session idle-delay "$SAVE_IDLE" 2>/dev/null || true
  [ -n "$SAVE_SLEEP_AC" ] && gsettings set org.gnome.settings-daemon.plugins.power sleep-inactive-ac-type "$SAVE_SLEEP_AC" 2>/dev/null || true
  [ -n "$SAVE_SLEEP_BAT" ] && gsettings set org.gnome.settings-daemon.plugins.power sleep-inactive-battery-type "$SAVE_SLEEP_BAT" 2>/dev/null || true
}
trap restore_screen EXIT

echo "===== 性能对比实验 ====="
echo "数据 ${USERS}用户/${VIDEOS}视频 · 常规 ${CONC}并发/${DUR} · 极端 ${EXTREME_CONC}并发/${EXTREME_DUR} · 每组 ${ROUNDS} 次"
echo "优化版 $OPT_URL · 退化版 $BASE_URL"

# 1. 构建
echo "== 构建 =="
(cd ../feed && go build -o /tmp/feed-opt .)
(cd ../feed-baseline && go build -o /tmp/feed-base .)
(cd bench && go build -o bench .)

# 2. 首次灌数据（-reset 清库 + FLUSHDB Redis）
echo "== seed 首次灌数据（重置）=="
(cd seed && go run . -reset -users "$USERS" -videos "$VIDEOS")

# 3. 启动两版
echo "== 启动两版 =="
pkill -f /tmp/feed-opt 2>/dev/null || true
pkill -f /tmp/feed-base 2>/dev/null || true
nohup /tmp/feed-opt > /tmp/feed-opt.log 2>&1 &
nohup /tmp/feed-base > /tmp/feed-base.log 2>&1 &
sleep 2
curl -s "$OPT_URL/video/detail?videoId=1" | grep -q '"code":0' && echo "  优化版 OK"
curl -s "$BASE_URL/video/detail?videoId=1" | grep -q '"code":0' && echo "  退化版 OK"

# 4. 跑一组：对两版各跑 ROUNDS 次，每次之间 seed 重置
run_group() {
  local scenario=$1 conc=$2 dur=$3
  echo ""
  echo "===== 场景 $scenario · 并发 $conc · $dur ====="
  for v in opt base; do
    local url=$OPT_URL
    [ "$v" = "base" ] && url=$BASE_URL
    for r in $(seq 1 "$ROUNDS"); do
      echo "-- [$v] $scenario 第 $r/$ROUNDS 次"
      (cd bench && ./bench -url "$url" -scenario "$scenario" -concurrency "$conc" \
        -duration "$dur" -version "$v" -token-file ../tokens.txt -notes "$NOTES" | tail -3)
      # 保留每次结果，避免覆盖
      mv results/result-${v}-${scenario}.json results/result-${v}-${scenario}-r${r}.json
      # 数据污染重置（同一 random-seed 保证 3 次数据一致）
      (cd seed && go run . -reset -users "$USERS" -videos "$VIDEOS" >/dev/null)
    done
  done
}

# 5. 全部实验组（任务书 §4.2）
run_group mix          "$CONC"        "$DUR"
run_group detail_hot   "$CONC"        "$DUR"   # A. 缓存价值
run_group detail_cold  "$CONC"        "$DUR"   # A'. 缓存 miss 对照
run_group detail_miss  "$CONC"        "$DUR"   # A''. 缓存穿透（空值缓存 vs 每次打库）
run_group reportPlay   "$CONC"        "$DUR"   # B. 计数优化
run_group list         "$CONC"        "$DUR"   # C. N+1 → 批量
run_group feed         "$CONC"        "$DUR"   # D. feed 存储
run_group mix          "$EXTREME_CONC" "$EXTREME_DUR"  # 极端组

# 5.5 写扩散压测（可选）：BIGV_FANS 个粉丝的大 V 上传，对比同步/异步扩散
if [ "$BIGV_FANS" -gt 0 ]; then
  echo ""
  echo "===== 写扩散压测（${BIGV_FANS} 粉丝大 V）====="
  (cd seed && go run . -reset -users "$USERS" -videos "$VIDEOS" -bigv-fans "$BIGV_FANS" >/dev/null)
  ./scripts/fanout_test.sh "$BASE_URL" 3
  ./scripts/fanout_test.sh "$OPT_URL" 3
fi

# 6. 收尾
pkill -f /tmp/feed-opt 2>/dev/null || true
pkill -f /tmp/feed-base 2>/dev/null || true
echo ""
echo "===== 实验完成，结果在 results/ ====="
echo "对比时：取每组 3 次的中位数（去掉最高最低）"
echo "下一步：把两版 result 汇总成对比表写入 ../feed/docs/性能对比报告.md"
