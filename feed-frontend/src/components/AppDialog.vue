<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div v-if="modelValue" class="dialog-mask" @click.self="onCancel">
        <div class="dialog">
          <h3 class="dialog-title">{{ title }}</h3>
          <p v-if="message" class="dialog-message">{{ message }}</p>
          <div class="dialog-actions">
            <button type="button" class="btn-cancel" @click="onCancel">取消</button>
            <button type="button" class="btn-confirm" @click="onConfirm">确定</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '提示' },
  message: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

function close() {
  emit('update:modelValue', false)
}

function onCancel() {
  close()
  emit('cancel')
}

function onConfirm() {
  close()
  emit('confirm')
}
</script>

<style scoped>
.dialog-mask {
  position: fixed;
  inset: 0;
  z-index: 9990;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: 80%;
  max-width: 320px;
  background: #fff;
  border-radius: var(--radius-card);
  padding: 24px 20px 16px;
  text-align: center;
}

.dialog-title {
  font-size: var(--font-md);
  font-weight: 600;
}

.dialog-message {
  margin-top: 8px;
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
}

.dialog-actions {
  display: flex;
  gap: 12px;
  margin-top: 20px;
}

.btn-cancel,
.btn-confirm {
  flex: 1;
  height: 42px;
  border-radius: var(--radius-btn);
  font-size: var(--font-md);
}

.btn-cancel {
  background: var(--color-input-bg);
  color: var(--color-text);
}

.btn-confirm {
  background: var(--color-primary);
  color: #fff;
}

.dialog-enter-active,
.dialog-leave-active {
  transition: opacity 0.2s;
}

.dialog-enter-active .dialog,
.dialog-leave-active .dialog {
  transition: transform 0.2s;
}

.dialog-enter-from,
.dialog-leave-to {
  opacity: 0;
}

.dialog-enter-from .dialog,
.dialog-leave-to .dialog {
  transform: scale(0.92);
}
</style>
