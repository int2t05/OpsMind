<template>
  <n-modal :show="visible" preset="dialog" :title="title" positive-text="确认" negative-text="取消"
    @positive-click="$emit('confirm')" @negative-click="$emit('cancel')" @close="$emit('cancel')">
    <p>{{ message }}</p>
  </n-modal>
</template>

<script setup lang="ts">
import { NModal } from 'naive-ui'

defineProps<{ visible: boolean; title: string; message: string }>()
defineEmits<{ (e: 'confirm'): void; (e: 'cancel'): void }>()
</script>
