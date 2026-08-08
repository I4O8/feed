<template>
  <div
    class="app-avatar"
    :style="{ width: size + 'px', height: size + 'px', fontSize: (size / 2.8) + 'px' }"
  >
    <img v-if="src" :src="src" :alt="name || '头像'" />
    <span v-else class="placeholder">{{ (name || '?').charAt(0) }}</span>
  </div>
</template>

<script setup>
defineProps({
  src: { type: String, default: '' },
  name: { type: String, default: '' },
  size: { type: Number, default: 64 },
})
</script>

<style scoped>
.app-avatar {
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: #e4e4e6;
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.placeholder {
  color: #8a8b91;
  font-weight: 600;
}
</style>
