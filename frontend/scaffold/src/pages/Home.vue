<script setup lang="ts">
// Reference overview page: stat tiles, charts, then one card per block.
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import Chart from '@/base/Chart.vue'
import HoverCard from '@/base/HoverCard.vue'
import Icon from '@/base/Icon.vue'
import Loading from '@/base/Loading.vue'
import ProgressBar from '@/base/ProgressBar.vue'
import { fmtRelative } from '@/base/time'
import { useFetch } from '@/base/useFetch'
import { exampleStats, listExamples } from '@/api/example'

const base = { q: '', status: '', range: '', sort: '', page: 1, size: 20 }

const summary = useFetch(
  async (signal) => {
    // the default order is newest first, so the unfiltered page doubles as "recent"
    const [all, enabled, today] = await Promise.all([
      listExamples(base, signal),
      listExamples({ ...base, status: 'enabled' }, signal),
      listExamples({ ...base, range: 'today' }, signal),
    ])
    return { total: all.total, enabled: enabled.total, today: today.total, recent: all.items.slice(0, 8) }
  },
  { immediate: true },
)
const stats = useFetch((signal) => exampleStats(14, signal), { immediate: true })

const trend = computed(() => {
  const days = stats.data.value ?? []
  return {
    grid: { left: 32, right: 8, top: 16, bottom: 24 },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: days.map((d) => d.date.slice(5)) },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{ name: '新增', type: 'bar', barMaxWidth: 24, data: days.map((d) => d.created) }],
  }
})

const split = computed(() => {
  const s = summary.data.value
  return {
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        type: 'pie',
        radius: ['50%', '72%'],
        top: -16,
        label: { show: false },
        data: s
          ? [
              { name: '已启用', value: s.enabled },
              { name: '已停用', value: s.total - s.enabled },
            ]
          : [],
      },
    ],
  }
})
</script>

<template>
  <div class="page">
    <Loading v-if="!summary.loaded.value" />
    <p v-else-if="summary.error.value" class="text-danger">加载失败:{{ summary.error.value }}</p>
    <template v-else-if="summary.data.value">
      <div class="stats">
        <div class="card">
          <span class="stat-label">示例总数</span>
          <span class="stat-value">{{ summary.data.value.total }}</span>
        </div>
        <div class="card">
          <span class="stat-label">已启用</span>
          <span class="stat-value">{{ summary.data.value.enabled }}</span>
          <ProgressBar :value="summary.data.value.enabled" :max="summary.data.value.total" label="启用比例" />
        </div>
        <div class="card">
          <span class="stat-label">
            今日更新
            <HoverCard>
              <button type="button" class="ghost btn-icon btn-sm" aria-label="统计口径">
                <Icon name="info" :size="14" />
              </button>
              <template #content>今天 0 点以来有过修改的示例数，包括新建和启停。按服务器时区计算。</template>
            </HoverCard>
          </span>
          <span class="stat-value">{{ summary.data.value.today }}</span>
        </div>
      </div>

      <div class="charts">
        <section class="card">
          <div class="card-head"><h3>近 14 天新增</h3></div>
          <Chart
            :option="trend"
            :loading="!stats.loaded.value"
            :empty="stats.loaded.value && !stats.data.value?.some((d) => d.created)"
            empty-text="近 14 天没有新增"
          />
        </section>
        <section class="card">
          <div class="card-head"><h3>状态分布</h3></div>
          <Chart :option="split" :empty="summary.data.value.total === 0" />
        </section>
      </div>

      <section class="card">
        <div class="card-head">
          <div>
            <h3>最近更新</h3>
            <p class="card-desc">最近修改的示例</p>
          </div>
          <RouterLink to="/examples" class="btn ghost btn-sm">全部</RouterLink>
        </div>
        <p v-if="summary.data.value.recent.length === 0" class="text-muted">暂无记录</p>
        <dl v-else class="kv">
          <template v-for="r in summary.data.value.recent" :key="r.id">
            <dt>{{ fmtRelative(r.updated_at) }}</dt>
            <dd>
              <RouterLink :to="`/examples/${r.id}`">{{ r.name }}</RouterLink>
            </dd>
          </template>
        </dl>
      </section>
    </template>
  </div>
</template>

<style scoped>
.charts {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
  gap: var(--space-3);
}
.stat-label {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}
.stat-label button {
  margin: -6px 0;
}
@media (max-width: 960px) {
  .charts {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
