<template>
  <div class="page auth-page">
    <header class="brand">
      <h1 class="title">注册</h1>
      <p class="muted">注册即登录，昵称稍后可在编辑资料里修改</p>
    </header>

    <form class="form" @submit.prevent="handleSubmit">
      <AppInput
        v-model="account"
        placeholder="手机号 / 邮箱"
        :error="errors.account"
        @update:model-value="onAccountInput"
      />
      <AppInput
        v-model="password"
        type="password"
        placeholder="密码（6-32 位）"
        :error="errors.password"
        @update:model-value="onPasswordInput"
      />
      <AppInput
        v-model="confirmPassword"
        type="password"
        placeholder="确认密码"
        :error="errors.confirm"
        @update:model-value="onConfirmInput"
      />
      <AppButton :loading="loading" class="submit" @click="handleSubmit">
        {{ loading ? '注册中...' : '注册' }}
      </AppButton>
    </form>

    <p class="switch-link">
      已有账号？
      <RouterLink to="/login">去登录</RouterLink>
    </p>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import AppInput from '../components/AppInput.vue'
import AppButton from '../components/AppButton.vue'
import { useUserStore } from '../stores/user'
import { validateAccount, isPasswordValid } from '../utils/validators'
import { showToast } from '../utils/toast'

const router = useRouter()
const userStore = useUserStore()

const account = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const errors = reactive({ account: '', password: '', confirm: '' })

// 输入时实时校验（打字即反馈）
function onAccountInput(v) {
  account.value = v
  errors.account = validateAccount(v)
}

function onPasswordInput(v) {
  password.value = v
  errors.password = v ? '' : '请输入密码'
  if (confirmPassword.value) {
    errors.confirm = confirmPassword.value === v ? '' : '两次输入的密码不一致'
  }
}

function onConfirmInput(v) {
  confirmPassword.value = v
  errors.confirm = v === password.value ? '' : '两次输入的密码不一致'
}

function validate() {
  if (!account.value) errors.account = '请输入手机号或邮箱'
  if (!password.value) errors.password = '请输入密码'
  else if (!isPasswordValid(password.value)) errors.password = '密码长度需为 6-32 位'
  if (confirmPassword.value !== password.value) errors.confirm = '两次输入的密码不一致'
  return !errors.account && !errors.password && !errors.confirm
}

async function handleSubmit() {
  if (loading.value) return
  if (!validate()) return

  loading.value = true
  try {
    await userStore.register(account.value.trim(), password.value)
    showToast('注册成功，已自动登录', 'success')
    router.push('/')
  } catch (err) {
    showToast(err?.message || '注册失败', 'error')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  justify-content: center;
  padding: 0 var(--space-page);
}

.brand {
  margin-bottom: 40px;
  text-align: center;
}

.title {
  font-size: var(--font-xl);
  font-weight: 700;
  color: var(--color-primary);
}

.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.submit {
  margin-top: 10px;
}

.switch-link {
  margin-top: 24px;
  text-align: center;
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
}

.switch-link a {
  color: var(--color-primary);
  font-weight: 600;
}
</style>
