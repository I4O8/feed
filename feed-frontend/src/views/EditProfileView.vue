<template>
  <div class="page edit-page">
    <header class="topbar">
      <button type="button" class="back" @click="router.back()">←</button>
      <span class="topbar-title">编辑资料</span>
      <span class="topbar-spacer"></span>
    </header>

    <div v-if="loading" class="loading muted">加载中...</div>

    <div v-else class="form">
      <!-- 头像区：上传即生效 -->
      <div class="avatar-row">
        <AppAvatar :src="avatarPreview" :name="form.name" :size="88" />
        <button type="button" class="change-avatar" :disabled="uploading" @click="fileInput.click()">
          {{ uploading ? '上传中...' : '更换头像' }}
        </button>
        <input
          ref="fileInput"
          type="file"
          accept="image/jpeg,image/png,image/webp"
          hidden
          @change="onFileChange"
        />
      </div>

      <div class="field">
        <label class="field-label">昵称</label>
        <AppInput v-model="form.name" maxlength="12" :error="errors.name" />
      </div>

      <div class="field">
        <label class="field-label">简介</label>
        <textarea
          v-model="form.bio"
          class="bio-input"
          maxlength="200"
          rows="4"
          placeholder="介绍一下自己"
        ></textarea>
        <p class="muted count">{{ form.bio.length }}/200</p>
      </div>

      <AppButton :loading="saving" class="save" @click="handleSave">
        {{ saving ? '保存中...' : '保存' }}
      </AppButton>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppAvatar from '../components/AppAvatar.vue'
import AppInput from '../components/AppInput.vue'
import AppButton from '../components/AppButton.vue'
import { useUserStore } from '../stores/user'
import { getAssetUrl } from '../api/asset'
import { showToast } from '../utils/toast'

const router = useRouter()
const userStore = useUserStore()

const loading = ref(true)
const saving = ref(false)
const uploading = ref(false)
const fileInput = ref(null)

const form = reactive({ name: '', bio: '' })
const errors = reactive({ name: '' })
const avatarPreview = ref('')

onMounted(async () => {
  loading.value = true
  try {
    const p = await userStore.fetchProfile(userStore.userId)
    form.name = p.name || ''
    form.bio = p.bio || ''
    avatarPreview.value = p.avatar ? getAssetUrl(p.avatar) : ''
  } catch (err) {
    showToast(err?.message || '加载失败', 'error')
  } finally {
    loading.value = false
  }
})

// 头像上传：前端预检类型/大小 → uploadAvatar → 即时刷新预览
async function onFileChange(e) {
  const file = e.target.files?.[0]
  if (!file) return

  const okTypes = ['image/jpeg', 'image/png', 'image/webp']
  if (!okTypes.includes(file.type)) {
    showToast('仅支持 jpg / png / webp 图片', 'error')
    e.target.value = ''
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    showToast('图片大小不能超过 5MB', 'error')
    e.target.value = ''
    return
  }

  uploading.value = true
  try {
    const avatarUrl = await userStore.uploadAvatar(file)
    avatarPreview.value = getAssetUrl(avatarUrl)
    showToast('头像已更新', 'success')
  } catch (err) {
    showToast(err?.message || '上传失败', 'error')
  } finally {
    uploading.value = false
    e.target.value = ''
  }
}

async function handleSave() {
  if (saving.value) return
  const name = form.name.trim()
  if (!name) {
    errors.name = '昵称不能为空'
    return
  }

  saving.value = true
  try {
    await userStore.updateProfile({ name, bio: form.bio.trim() })
    showToast('保存成功', 'success')
    router.back()
  } catch (err) {
    showToast(err?.message || '保存失败', 'error')
  } finally {
    saving.value = false
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

.loading {
  padding: 48px 0;
  text-align: center;
}

.avatar-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  margin: 24px 0 32px;
}

.change-avatar {
  font-size: var(--font-sm);
  color: var(--color-primary);
  font-weight: 600;
}

.change-avatar:disabled {
  opacity: 0.6;
}

.field {
  margin-bottom: 20px;
}

.field-label {
  display: block;
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.bio-input {
  width: 100%;
  padding: 12px 16px;
  border-radius: var(--radius-input);
  background: var(--color-input-bg);
  font-size: var(--font-md);
  color: var(--color-text);
  border: 1px solid transparent;
  resize: none;
  transition: border-color 0.2s;
}

.bio-input:focus {
  border-color: var(--color-primary);
  background: #fff;
}

.count {
  margin-top: 6px;
  text-align: right;
}

.save {
  margin-top: 8px;
}
</style>
