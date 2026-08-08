<template>
  <div class="page publish-page">
    <header class="topbar">
      <button type="button" class="back" @click="router.back()">←</button>
      <span class="topbar-title">发布视频</span>
      <span class="topbar-spacer"></span>
    </header>

    <div class="form">
      <!-- 视频选择 / 预览 -->
      <div v-if="!videoFile" class="pick-area" @click="videoInput.click()">
        <span class="pick-icon">＋</span>
        <span class="pick-text">选择视频（mp4，≤100MB）</span>
      </div>
      <div v-else class="video-preview">
        <VideoPlayer :src="videoPreviewUrl" muted loop :controls="true" fit="contain" />
        <button type="button" class="re-pick" @click="removeVideo">重新选择</button>
      </div>
      <input ref="videoInput" type="file" accept="video/mp4" hidden @change="onSelectVideo" />

      <div class="field">
        <label class="field-label">标题（必填）</label>
        <AppInput v-model="title" maxlength="30" placeholder="添加视频标题" :error="errors.title" />
      </div>

      <div class="field">
        <label class="field-label">描述（可选）</label>
        <textarea
          v-model="describe"
          class="bio-input"
          maxlength="200"
          rows="3"
          placeholder="介绍一下这个视频"
        ></textarea>
      </div>

      <!-- 封面（可选） -->
      <div class="field">
        <label class="field-label">封面（可选，不选将自动截取视频一帧）</label>
        <div class="cover-row">
          <img v-if="coverPreviewUrl" :src="coverPreviewUrl" class="cover-thumb" alt="封面预览" />
          <button type="button" class="cover-btn" @click="coverInput.click()">
            {{ coverPreviewUrl ? '更换封面' : '添加封面' }}
          </button>
        </div>
        <input
          ref="coverInput"
          type="file"
          accept="image/jpeg,image/png,image/webp"
          hidden
          @change="onSelectCover"
        />
      </div>

      <AppButton class="publish-btn" :loading="uploading" @click="handlePublish">
        <template v-if="uploading">上传中 {{ progress }}%</template>
        <template v-else>发布</template>
      </AppButton>
    </div>
  </div>
</template>

<script setup>
import { onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppInput from '../components/AppInput.vue'
import AppButton from '../components/AppButton.vue'
import VideoPlayer from '../components/VideoPlayer.vue'
import { uploadVideo } from '../api/video'
import { showToast } from '../utils/toast'

const router = useRouter()

const videoInput = ref(null)
const coverInput = ref(null)
const videoFile = ref(null)
const videoPreviewUrl = ref('')
const coverFile = ref(null)
const coverPreviewUrl = ref('')
const title = ref('')
const describe = ref('')
const uploading = ref(false)
const progress = ref(0)
const errors = reactive({ title: '' })

const VIDEO_MAX = 100 * 1024 * 1024 // 100MB
const COVER_MAX = 5 * 1024 * 1024 // 5MB
const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp']

function onSelectVideo(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (file.type !== 'video/mp4') {
    showToast('仅支持 mp4 视频', 'error')
    return
  }
  if (file.size > VIDEO_MAX) {
    showToast('视频不能超过 100MB', 'error')
    return
  }
  videoFile.value = file
  videoPreviewUrl.value = URL.createObjectURL(file)
}

function removeVideo() {
  if (videoPreviewUrl.value) URL.revokeObjectURL(videoPreviewUrl.value)
  videoFile.value = null
  videoPreviewUrl.value = ''
}

function onSelectCover(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!IMAGE_TYPES.includes(file.type)) {
    showToast('封面仅支持 jpg / png / webp', 'error')
    return
  }
  if (file.size > COVER_MAX) {
    showToast('封面不能超过 5MB', 'error')
    return
  }
  if (coverPreviewUrl.value) URL.revokeObjectURL(coverPreviewUrl.value)
  coverFile.value = file
  coverPreviewUrl.value = URL.createObjectURL(file)
}

async function handlePublish() {
  if (uploading.value) return
  if (!videoFile.value) {
    showToast('请先选择视频', 'error')
    return
  }
  if (!title.value.trim()) {
    errors.title = '请输入标题'
    return
  }

  const formData = new FormData()
  formData.append('file', videoFile.value)
  formData.append('title', title.value.trim())
  if (describe.value.trim()) formData.append('describe', describe.value.trim())
  if (coverFile.value) formData.append('cover', coverFile.value)

  uploading.value = true
  progress.value = 0
  try {
    const data = await uploadVideo(formData, (p) => {
      progress.value = p
    })
    showToast('发布成功', 'success')
    router.replace(`/video/${data.videoId}`)
  } catch (err) {
    showToast(err?.message || '发布失败', 'error')
  } finally {
    uploading.value = false
  }
}

onUnmounted(() => {
  if (videoPreviewUrl.value) URL.revokeObjectURL(videoPreviewUrl.value)
  if (coverPreviewUrl.value) URL.revokeObjectURL(coverPreviewUrl.value)
})
</script>

<style scoped>
.publish-page {
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
  margin-top: 20px;
}

.pick-area {
  height: 160px;
  border: 1.5px dashed var(--color-border);
  border-radius: var(--radius-card);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.pick-icon {
  font-size: 32px;
  color: var(--color-primary);
}

.pick-text {
  font-size: var(--font-sm);
}

.video-preview {
  position: relative;
  height: 220px;
  background: #000;
  border-radius: var(--radius-card);
  overflow: hidden;
}

.re-pick {
  position: absolute;
  bottom: 8px;
  right: 8px;
  font-size: var(--font-xs);
  padding: 4px 10px;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  border-radius: 999px;
}

.field {
  margin-top: 20px;
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

.cover-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.cover-thumb {
  width: 64px;
  height: 64px;
  border-radius: var(--radius-card);
  object-fit: cover;
}

.cover-btn {
  font-size: var(--font-sm);
  color: var(--color-primary);
  font-weight: 600;
}

.publish-btn {
  margin-top: 28px;
}
</style>
