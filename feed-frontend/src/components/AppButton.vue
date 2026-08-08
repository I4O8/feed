<template>
  <button
    type="button"
    class="app-btn"
    :class="{ 'is-primary': primary, 'is-ghost': !primary, 'is-loading': loading }"
    :disabled="loading || disabled"
    @click="$emit('click')"
  >
    <span v-if="loading" class="spinner" aria-hidden="true"></span>
    <slot />
  </button>
</template>

<script setup>
defineProps({
  primary: { type: Boolean, default: true },
  loading: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
})

defineEmits(['click'])
</script>

<style scoped>
.app-btn {
  width: 100%;
  height: 48px;
  border-radius: var(--radius-btn);
  font-size: var(--font-md);
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: opacity 0.2s;
}

.is-primary {
  background: var(--color-primary);
  color: #fff;
}

.is-primary:active {
  opacity: 0.85;
}

.is-ghost {
  background: transparent;
  color: var(--color-primary);
  border: 1px solid var(--color-primary);
}

.app-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.4);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
