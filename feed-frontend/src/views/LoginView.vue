<template>
  <div class="page auth-page">
    <header class="brand">
      <h1 class="title">仿抖音</h1>
      <p class="muted">登录你的账号</p>
    </header>

    <form class="form" @submit.prevent="handleSubmit">
      <AppInput
        v-model="account"
        placeholder="手机号 / 邮箱"
        :error="errors.account"
      />
      <AppInput
        v-model="password"
        type="password"
        placeholder="密码"
        :error="errors.password"
      />
      <AppButton :loading="loading" class="submit" @click="handleSubmit">
        {{ loading ? '登录中...' : '登录' }}
      </AppButton>
    </form>

    <p class="switch-link">
      还没有账号？
      <RouterLink to="/register">去注册</RouterLink>
    </p>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import AppInput from '../components/AppInput.vue'
import AppButton from '../components/AppButton.vue'
import { useUserStore } from '../stores/user'
import { showToast } from '../utils/toast'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const account = ref('')
const password = ref('')
const loading = ref(false)
const errors = reactive({ account: '', password: '' })

function validate() {
  errors.account = account.value ? '' : '请输入手机号或邮箱'
  errors.password = password.value ? '' : '请输入密码'
  return !errors.account && !errors.password
}

async function handleSubmit() {
  if (loading.value) return
  if (!validate()) return

  loading.value = true
  try {
    await userStore.login(account.value.trim(), password.value)
    showToast('登录成功', 'success')
    // 优先回跳来源页（如之前被拦截的页面），否则进首页
    const redirect = route.query.redirect
    router.push(redirect ? String(redirect) : '/')
  } catch (err) {
    showToast(err?.message || '登录失败', 'error')
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
