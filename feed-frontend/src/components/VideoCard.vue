<template>
  <div class="video-card">
    <div class="cover-wrap" @click="$emit('open')">
      <img
        v-if="video.coverUrl"
        :src="getAssetUrl(video.coverUrl)"
        class="cover"
        alt="封面"
        loading="lazy"
      />
      <div v-else class="cover-placeholder">▶</div>
      <span class="duration">{{ formatDuration(video.duration) }}</span>
      <button
        v-if="showDelete"
        type="button"
        class="del"
        title="删除"
        @click.stop="$emit('delete')"
      >×</button>
    </div>
    <p class="title" @click="$emit('open')">{{ video.title }}</p>
    <p class="meta">▶ {{ formatCount(video.playCount) }}</p>
  </div>
</template>

<script setup>
import { getAssetUrl } from '../api/asset'
import { formatCount, formatDuration } from '../utils/format'

defineProps({
  video: { type: Object, required: true },
  showDelete: { type: Boolean, default: false },
})

defineEmits(['open', 'delete'])
</script>

<style scoped>
.video-card {
  overflow: hidden;
}

.cover-wrap {
  position: relative;
  aspect-ratio: 9 / 16;
  border-radius: var(--radius-card);
  overflow: hidden;
  background: #000;
  cursor: pointer;
}

.cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cover-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.7);
  font-size: 32px;
}

.duration {
  position: absolute;
  right: 6px;
  bottom: 6px;
  font-size: 11px;
  color: #fff;
  background: rgba(0, 0, 0, 0.6);
  padding: 1px 6px;
  border-radius: 4px;
}

.del {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  font-size: 14px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.title {
  margin-top: 6px;
  font-size: var(--font-sm);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta {
  margin-top: 2px;
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
}
</style>
