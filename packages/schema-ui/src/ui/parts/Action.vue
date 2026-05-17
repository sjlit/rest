<template>
  <el-button
    :type="action.type || 'default'"
    :size="action.size || 'small'"
    :round="action.round"
    :icon="action.icon"
    :loading="loading"
    @click="handleClick"
  >
    {{ action.label }}
  </el-button>
</template>

<script setup lang="ts">
import { ref } from 'vue'
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

function handleClick() {
  if (props.action.hidden) {
    if (typeof props.action.hidden === 'function') {
      const result = props.action.hidden(props.model || {})
      if (result instanceof Promise) {
        result.then((hidden) => {
          if (!hidden) emit('click', props.action, loading)
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
