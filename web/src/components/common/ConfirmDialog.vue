<template>
  <n-modal :show="visible" preset="dialog" :title="title" positive-text="确认" negative-text="取消"
    @positive-click="$emit('confirm')" @negative-click="$emit('cancel')" @close="$emit('cancel')">
    <p>{{ message }}</p>
  </n-modal>
</template>

<script setup lang="ts">
// TODO(components/ConfirmDialog): 此组件未被任何视图或组件 import — 属于死代码。
//                              要么在需要确认对话框的页面中统一使用本组件替代内联模态框，要么删除。
import { NModal } from 'naive-ui'

defineProps<{ visible: boolean; title: string; message: string }>()
defineEmits<{ (e: 'confirm'): void; (e: 'cancel'): void }>()
</script>
