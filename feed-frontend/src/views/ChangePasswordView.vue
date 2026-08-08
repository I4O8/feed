<template>
  <div class="page edit-page">
    <header class="topbar">
      <button type="button" class="back" @click="router.back()">←</button>
      <span class="topbar-title">修改密码</span>
      <span class="topbar-spacer"></span>
    </header>

    <form class="form" @submit.prevent="handleSubmit">
      <AppInput
        v-model="oldPassword"
        type="password"
        placeholder="旧密码"
        :error="errors.old"
      />
      <AppInput
        v-model="newPassword"
        type="password"
        placeholder="新密码（6-32 位）"
        :error="errors.new"
        @update:model-value="onNewInput"
      />
      <AppInput
        v-model="confirmPassword"
        type="password"
        placeholder="确认新密码"
        :error="errors.confirm"
        @update:model-value="onConfirmInput"
      />
      <AppButton :loading="loading" class="save" @click="handleSubmit">
        {{ loading ? '提交中...' : '确认修改' }}
      </AppButton>
    </form>

    <p class="muted tip">修改成功后无需重新登录，当前登录状态保持有效</p>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppInput from '../components/AppInput.vue'
import AppButton from '../components/AppButton.vue'
import { useUserStore } from '../stores/user'
import { isPasswordValid } from '../utils/validators'
import { showToast } from '../utils/toast'

const router = useRouter()
const userStore = useUserStore()

const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const errors = reactive({ old: '', new: '', confirm: '' })

function onNewInput(v) {
  newPassword.value = v
  errors.new = v ? '' : '请输入新密码'
  if (confirmPassword.value) {
    errors.confirm = confirmPassword.value === v ? '' : '两次输入的密码不一致'
  }
}

function onConfirmInput(v) {
  confirmPassword.value = v
  errors.confirm = v === newPassword.value ? '' : '两次输入的密码不一致'
}

function validate() {
  errors.old = oldPassword.value ? '' : '请输入旧密码'
  if (!newPassword.value) errors.new = '请输入新密码'
  else if (!isPasswordValid(newPassword.value)) errors.new = '密码长度需为 6-32 位'
  else if (newPassword.value === oldPassword.value) errors.new = '新密码不能与旧密码相同'
  if (confirmPassword.value !== newPassword.value) errors.confirm = '两次输入的密码不一致'
  return !errors.old && !errors.new && !errors.confirm
}

async function handleSubmit() {
  if (loading.value) return
  if (!validate()) return

  loading.value = true
  try {
    await userStore.changePassword({
      oldPassword: oldPassword.value,
      newPassword: newPassword.value,
    })
    showToast('修改成功', 'success')
    router.back()
  } catch (err) {
    showToast(err?.message || '修改失败', 'error')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.edit-page {
  padding: 0 var(--space-page) 32px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  margin: 0 calc(-1 * var(--space-page));
  padding: 0 var(--space-page);
  border-bottom: 1px solid var(--color-border);
}

.back {
  font-size: 20px;
  width: 32px;
  text-align: left;
  color: var(--color-text);
}

.topbar-title {
  font-size: var(--font-md);
  font-weight: 600;
}

.topbar-spacer {
  width: 32px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 24px;
}

.save {
  margin-top: 10px;
}

.tip {
  margin-top: 16px;
  text-align: center;
}
</style>
