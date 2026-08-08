<template>
  <Transition name="toast">
    <div v-if="toastState.visible" class="app-toast" :class="toastState.type">
      {{ toastState.message }}
    </div>
  </Transition>
</template>

<script setup>
import { toastState } from '../utils/toast'
</script>

<style scoped>
.app-toast {
  position: fixed;
  top: 48px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  max-width: 80%;
  padding: 10px 18px;
  border-radius: 8px;
  font-size: var(--font-sm);
  color: #fff;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.16);
  text-align: center;
}

.app-toast.info {
  background: rgba(22, 24, 35, 0.92);
}

.app-toast.success {
  background: var(--color-success);
}

.app-toast.error {
  background: var(--color-danger);
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.25s, transform 0.25s;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-8px);
}
</style>
