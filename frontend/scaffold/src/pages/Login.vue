<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { app } from '@/base/app'
import { errorMessage } from '@/base/errors'
import { useFormValidation } from '@/base/useFormValidation'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const form = reactive({ identifier: '', password: '' })
const submitting = ref(false)
const failure = ref('')

const v = useFormValidation(() => ({
  identifier: form.identifier.trim() ? '' : '请输入用户名或邮箱',
  password: form.password ? '' : '请输入密码',
}))

async function submit() {
  if (!v.validate() || submitting.value) return
  submitting.value = true
  failure.value = ''
  try {
    await auth.login(form.identifier.trim(), form.password)
    const next = typeof route.query.next === 'string' && route.query.next.startsWith('/') ? route.query.next : '/'
    await router.replace(next)
  } catch (e) {
    failure.value = errorMessage(e)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login">
    <form class="card form login-card" novalidate v-on="v.form" @submit.prevent="submit">
      <h1>{{ app.name }}</h1>
      <label class="field">
        <span class="field-label">用户名或邮箱</span>
        <input
          v-model="form.identifier"
          name="identifier"
          autocomplete="username"
          autofocus
          :aria-invalid="!!v.error('identifier')"
        />
        <span v-if="v.error('identifier')" class="field-error">{{ v.error('identifier') }}</span>
      </label>
      <label class="field">
        <span class="field-label">密码</span>
        <input
          v-model="form.password"
          type="password"
          name="password"
          autocomplete="current-password"
          :aria-invalid="!!v.error('password')"
        />
        <span v-if="v.error('password')" class="field-error">{{ v.error('password') }}</span>
      </label>
      <p v-if="failure" class="field-error" role="alert">{{ failure }}</p>
      <button type="submit" class="primary" :disabled="submitting">{{ submitting ? '登录中' : '登录' }}</button>
    </form>
  </div>
</template>

<style scoped>
.login {
  display: grid;
  place-items: center;
  min-height: 100dvh;
  padding: var(--space-4);
}
.login-card {
  width: min(360px, 100%);
  padding: var(--space-6);
}
</style>
