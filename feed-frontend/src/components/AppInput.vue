<template>
  <div class="app-input" :class="{ 'has-error': !!error }">
    <div class="field">
      <input
        :type="innerType"
        :value="modelValue"
        :placeholder="placeholder"
        :maxlength="maxlength"
        autocomplete="off"
        @input="$emit('update:modelValue', $event.target.value)"
      />
      <button
        v-if="type === 'password'"
        type="button"
        class="toggle"
        @click="visible = !visible"
      >
        {{ visible ? '隐藏' : '显示' }}
      </button>
    </div>
    <p v-if="error" class="error-msg">{{ error }}</p>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  type: { type: String, default: 'text' }, // text | password
  placeholder: { type: String, default: '' },
  maxlength: { type: Number, default: undefined },
  error: { type: String, default: '' },
})

defineEmits(['update:modelValue'])

const visible = ref(false)
const innerType = computed(() =>
  props.type === 'password' && !visible.value ? 'password' : 'text'
)
</script>

<style scoped>
.app-input {
  margin-bottom: 4px;
}

.field {
  position: relative;
  display: flex;
  align-items: center;
}

.field input {
  width: 100%;
  height: 48px;
  padding: 0 16px;
  border-radius: var(--radius-input);
  background: var(--color-input-bg);
  font-size: var(--font-md);
  color: var(--color-text);
  border: 1px solid transparent;
  transition: border-color 0.2s;
}

.field input::placeholder {
  color: var(--color-text-secondary);
}

.field input:focus {
  border-color: var(--color-primary);
  background: #fff;
}

.toggle {
  position: absolute;
  right: 12px;
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
  padding: 4px;
}

.error-msg {
  margin-top: 6px;
  font-size: var(--font-xs);
  color: var(--color-danger);
}

.has-error .field input {
  border-color: var(--color-danger);
}
</style>
