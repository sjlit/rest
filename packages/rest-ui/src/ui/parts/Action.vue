<template>
  <el-tooltip v-if="isComponentIcon" :content="action.label">
    <el-icon class="action-icon" @click="handleClick">
      <component :is="props.action.icon" />
    </el-icon>
  </el-tooltip>
  <el-button v-else :type="action.type || 'default'" :size="action.size || 'default'" :round="action.round"
    :loading="loading" @click="handleClick">
    {{ action.label }}
  </el-button>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Action as ActionType, Model } from '../../core/types'

interface Props {
  action: ActionType
  model?: Model
}

const props = defineProps<Props>()
const emit = defineEmits<{
  click: [action: ActionType, loading: ReturnType<typeof ref<boolean>>]
}>()

const loading = ref(false)

const isComponentIcon = computed(() => props.action.icon !== undefined && props.action.icon !== null && typeof props.action.icon !== 'string')

function handleClick() {
  if (props.action.hidden) {
    if (typeof props.action.hidden === 'function') {
      const result = props.action.hidden(props.model || {})
      if (result instanceof Promise) {
        result
          .then((hidden) => {
            if (!hidden) emit('click', props.action, loading)
          })
          .catch((err) => {
            console.error('Action hidden check failed:', err)
            emit('click', props.action, loading)
          })
        return
      }
      if (result) return
    } else if (props.action.hidden) {
      return
    }
  }
  emit('click', props.action, loading)
}
</script>
