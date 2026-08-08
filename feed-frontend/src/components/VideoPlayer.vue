<template>
  <video
    ref="videoEl"
    class="video-player"
    :class="`fit-${fit}`"
    :src="src"
    :poster="poster || undefined"
    :autoplay="autoplay"
    :muted="muted"
    :loop="loop"
    :controls="controls"
    :playsinline="playsinline"
    preload="metadata"
    @play="emit('play')"
    @pause="emit('pause')"
    @ended="emit('ended')"
    @timeupdate="onTimeupdate"
    @click="emit('click')"
  ></video>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  src: { type: String, default: '' },
  poster: { type: String, default: '' },
  autoplay: { type: Boolean, default: false },
  muted: { type: Boolean, default: false },
  loop: { type: Boolean, default: false },
  controls: { type: Boolean, default: false },
  playsinline: { type: Boolean, default: true },
  // 铺满 cover（首页沉浸） / 完整显示 contain（详情页）
  fit: { type: String, default: 'cover' },
})

const emit = defineEmits(['play', 'pause', 'ended', 'timeupdate', 'click'])

const videoEl = ref(null)

// 回传播放进度，供页面做 50% 上报判断
function onTimeupdate() {
  const v = videoEl.value
  if (!v) return
  emit('timeupdate', {
    currentTime: v.currentTime,
    duration: v.duration || 0,
    progress: v.duration ? v.currentTime / v.duration : 0,
  })
}

function play() {
  // 浏览器可能拒绝（自动播放策略），静默忽略，页面可回退处理
  videoEl.value?.play()?.catch(() => {})
}

function pause() {
  videoEl.value?.pause()
}

function toggle() {
  const v = videoEl.value
  if (!v) return
  if (v.paused) play()
  else pause()
}

// 首页手势后：取消静音并播放（恢复声音的一次用户手势）
function unmuteAndPlay() {
  const v = videoEl.value
  if (!v) return
  v.muted = false
  play()
}

function isMuted() {
  return videoEl.value?.muted ?? true
}

defineExpose({ play, pause, toggle, unmuteAndPlay, isMuted })
</script>

<style scoped>
.video-player {
  width: 100%;
  height: 100%;
  display: block;
  background: #000;
}

.fit-cover {
  object-fit: cover;
}

.fit-contain {
  object-fit: contain;
}
</style>
