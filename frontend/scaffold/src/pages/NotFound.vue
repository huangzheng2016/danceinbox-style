<script setup lang="ts">
import { RouterLink } from 'vue-router'
import EmptyState from '@/base/EmptyState.vue'
</script>

<template>
  <EmptyState icon="alert-triangle" title="页面不存在" hint="地址可能已变更，请从导航进入。">
    <RouterLink to="/" class="btn">返回概览</RouterLink>
  </EmptyState>
</template>
