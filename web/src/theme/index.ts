// Naive UI 主题覆盖配置 — 与 Linear Design 设计系统完全对齐。
//
// 为什么自定义主题而非使用默认 darkTheme：
// 默认 darkTheme 的色值与 Linear Design 有偏差（背景过亮、品牌色不同），
// 通过 themeOverrides 精确映射到 Linear 的色板，保证视觉一致性。

import type { GlobalThemeOverrides } from 'naive-ui'

// ===== 暗色主题覆盖（默认）=====
export const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#5e6ad2',
    primaryColorHover: '#7170ff',
    primaryColorPressed: '#4e5ab8',
    primaryColorSuppl: '#7170ff',
    infoColor: '#7170ff',
    infoColorHover: '#828fff',
    successColor: '#27a644',
    successColorHover: '#10b981',
    warningColor: '#f0a020',
    errorColor: '#e0484c',
    // 背景色
    bodyColor: '#08090a',
    cardColor: '#0f1011',
    modalColor: '#191a1b',
    popoverColor: '#191a1b',
    // 文字色
    textColorBase: '#f7f8f8',
    textColor1: '#f7f8f8',
    textColor2: '#d0d6e0',
    textColor3: '#8a8f98',
    // 边框
    borderColor: 'rgba(255,255,255,0.08)',
    dividerColor: 'rgba(255,255,255,0.05)',
    // 输入框
    inputColor: 'rgba(255,255,255,0.02)',
    // hover
    hoverColor: 'rgba(255,255,255,0.04)',
    // 滚动条
    scrollbarColor: 'rgba(255,255,255,0.1)',
    scrollbarColorHover: 'rgba(255,255,255,0.18)',
    // 阴影
    boxShadow1: '0 1px 2px rgba(0,0,0,0.3)',
    boxShadow2: '0 4px 12px rgba(0,0,0,0.4)',
    boxShadow3: '0 8px 24px rgba(0,0,0,0.5)',
  },
  Layout: {
    color: '#08090a',
    siderColor: '#0f1011',
    siderBorderColor: 'rgba(255,255,255,0.05)',
    headerColor: '#0f1011',
    headerBorderColor: 'rgba(255,255,255,0.05)',
    footerColor: '#08090a',
  },
  Menu: {
    itemColor: '#0f1011',
    itemColorHover: '#191a1b',
    itemTextColor: '#8a8f98',
    itemTextColorHover: '#f7f8f8',
    itemTextColorActive: '#f7f8f8',
    itemTextColorChildActive: '#f7f8f8',
    itemIconColor: '#8a8f98',
    itemIconColorHover: '#f7f8f8',
    itemIconColorActive: '#7170ff',
    itemColorActive: '#191a1b',
    itemColorActiveHover: '#191a1b',
    itemColorChildActive: 'rgba(94,106,210,0.12)',
    itemColorChildActiveHover: 'rgba(94,106,210,0.16)',
    arrowColor: '#62666d',
    arrowColorHover: '#d0d6e0',
    arrowColorActive: '#7170ff',
  },
  Button: {
    colorHover: 'rgba(255,255,255,0.06)',
    colorPressed: 'rgba(255,255,255,0.08)',
    colorFocus: 'rgba(255,255,255,0.04)',
    textColor: '#d0d6e0',
    textColorHover: '#f7f8f8',
    textColorPressed: '#f7f8f8',
    border: '1px solid rgba(255,255,255,0.08)',
    borderHover: '1px solid rgba(255,255,255,0.12)',
    borderPressed: '1px solid rgba(255,255,255,0.12)',
    borderFocus: '1px solid rgba(255,255,255,0.08)',
    borderRadius: '6px',
  },
  Input: {
    color: 'rgba(255,255,255,0.02)',
    colorFocus: 'rgba(255,255,255,0.04)',
    border: '1px solid rgba(255,255,255,0.08)',
    borderHover: '1px solid rgba(255,255,255,0.12)',
    borderFocus: '1px solid #5e6ad2',
    textColor: '#f7f8f8',
    placeholderColor: '#62666d',
    borderRadius: '6px',
  },
  Card: {
    color: 'rgba(255,255,255,0.02)',
    borderColor: 'rgba(255,255,255,0.08)',
    borderRadius: '8px',
    titleTextColor: '#f7f8f8',
    textColor: '#8a8f98',
  },
  Tag: {
    color: 'transparent',
    border: '1px solid rgba(255,255,255,0.08)',
    borderRadius: '9999px',
    textColor: '#d0d6e0',
  },
  DataTable: {
    tdColor: '#0f1011',
    tdColorHover: '#191a1b',
    thColor: '#0f1011',
    borderColor: 'rgba(255,255,255,0.05)',
  },
  Pagination: {
    itemColor: 'rgba(255,255,255,0.02)',
    itemColorHover: 'rgba(255,255,255,0.06)',
    itemColorActive: '#5e6ad2',
    itemTextColorActive: '#ffffff',
    itemBorder: '1px solid rgba(255,255,255,0.08)',
    borderRadius: '6px',
  },
  Modal: {
    color: '#191a1b',
    border: '1px solid rgba(255,255,255,0.08)',
    textColor: '#f7f8f8',
    borderRadius: '12px',
  },
  Select: {
    peers: {
      InternalSelection: {
        color: 'rgba(255,255,255,0.02)',
        border: '1px solid rgba(255,255,255,0.08)',
        borderHover: '1px solid rgba(255,255,255,0.12)',
        borderFocus: '1px solid #5e6ad2',
        borderRadius: '6px',
      },
    },
  },
  Switch: {
    railColorActive: '#5e6ad2',
  },
  Slider: {
    fillColor: '#5e6ad2',
    fillColorHover: '#7170ff',
  },
  Progress: {
    fillColor: '#5e6ad2',
  },
  Tabs: {
    tabTextColor: '#8a8f98',
    tabTextColorActive: '#f7f8f8',
    barColor: '#5e6ad2',
    tabColor: '#0f1011',
  },
  Breadcrumb: {
    itemTextColor: '#8a8f98',
    itemTextColorHover: '#f7f8f8',
    itemTextColorPressed: '#f7f8f8',
    separatorColor: '#62666d',
  },
  Badge: {
    color: '#e0484c',
  },
  Notification: {
    color: '#191a1b',
    border: '1px solid rgba(255,255,255,0.08)',
    textColor: '#f7f8f8',
    borderRadius: '12px',
  },
  Message: {
    color: '#191a1b',
    border: '1px solid rgba(255,255,255,0.08)',
    borderRadius: '8px',
  },
}

// ===== 浅色主题覆盖 =====
export const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#5e6ad2',
    primaryColorHover: '#7170ff',
    primaryColorPressed: '#4e5ab8',
    primaryColorSuppl: '#7170ff',
    infoColor: '#7170ff',
    infoColorHover: '#828fff',
    successColor: '#27a644',
    successColorHover: '#10b981',
    warningColor: '#f0a020',
    errorColor: '#e0484c',
    // 背景色（浅色）
    bodyColor: '#f7f8f8',
    cardColor: '#ffffff',
    modalColor: '#ffffff',
    popoverColor: '#ffffff',
    // 文字色（浅色）
    textColorBase: '#1a1a2e',
    textColor1: '#1a1a2e',
    textColor2: '#4a4a5a',
    textColor3: '#8a8f98',
    // 边框（浅色）
    borderColor: '#d0d6e0',
    dividerColor: '#e6e6e6',
    // 输入框（浅色）
    inputColor: '#ffffff',
    // hover（浅色）
    hoverColor: 'rgba(0,0,0,0.03)',
    // 滚动条
    scrollbarColor: 'rgba(0,0,0,0.12)',
    scrollbarColorHover: 'rgba(0,0,0,0.22)',
    // 阴影
    boxShadow1: '0 1px 2px rgba(0,0,0,0.06)',
    boxShadow2: '0 4px 12px rgba(0,0,0,0.08)',
    boxShadow3: '0 8px 24px rgba(0,0,0,0.12)',
  },
  Layout: {
    color: '#f7f8f8',
    siderColor: '#ffffff',
    siderBorderColor: '#e6e6e6',
    headerColor: '#ffffff',
    headerBorderColor: '#e6e6e6',
    footerColor: '#f7f8f8',
  },
  Menu: {
    itemColor: '#ffffff',
    itemColorHover: '#f3f4f5',
    itemTextColor: '#4a4a5a',
    itemTextColorHover: '#1a1a2e',
    itemTextColorActive: '#1a1a2e',
    itemIconColor: '#8a8f98',
    itemIconColorHover: '#1a1a2e',
    itemIconColorActive: '#5e6ad2',
    itemColorActive: '#f3f4f5',
    itemColorActiveHover: '#f3f4f5',
    arrowColor: '#8a8f98',
    arrowColorHover: '#4a4a5a',
    arrowColorActive: '#5e6ad2',
  },
  Button: {
    colorHover: 'rgba(0,0,0,0.04)',
    colorPressed: 'rgba(0,0,0,0.06)',
    colorFocus: 'rgba(0,0,0,0.02)',
    textColor: '#4a4a5a',
    textColorHover: '#1a1a2e',
    border: '1px solid #d0d6e0',
    borderHover: '1px solid #b0b6c0',
    borderPressed: '1px solid #b0b6c0',
    borderRadius: '6px',
  },
  Input: {
    color: '#ffffff',
    colorFocus: '#ffffff',
    border: '1px solid #d0d6e0',
    borderHover: '1px solid #b0b6c0',
    borderFocus: '1px solid #5e6ad2',
    textColor: '#1a1a2e',
    placeholderColor: '#8a8f98',
    borderRadius: '6px',
  },
  Card: {
    color: '#ffffff',
    borderColor: '#e6e6e6',
    borderRadius: '8px',
    titleTextColor: '#1a1a2e',
    textColor: '#4a4a5a',
  },
  Tag: {
    color: '#f3f4f5',
    border: '1px solid #e6e6e6',
    borderRadius: '9999px',
    textColor: '#4a4a5a',
  },
  DataTable: {
    tdColor: '#ffffff',
    tdColorHover: '#f7f8f8',
    thColor: '#f7f8f8',
    borderColor: '#e6e6e6',
  },
  Pagination: {
    itemColor: '#ffffff',
    itemColorHover: '#f3f4f5',
    itemColorActive: '#5e6ad2',
    itemBorder: '1px solid #d0d6e0',
    borderRadius: '6px',
  },
  Modal: {
    color: '#ffffff',
    border: '1px solid #e6e6e6',
    textColor: '#1a1a2e',
    borderRadius: '12px',
  },
  Select: {
    peers: {
      InternalSelection: {
        color: '#ffffff',
        border: '1px solid #d0d6e0',
        borderHover: '1px solid #b0b6c0',
        borderFocus: '1px solid #5e6ad2',
        borderRadius: '6px',
      },
    },
  },
  Switch: {
    railColorActive: '#5e6ad2',
  },
  Slider: {
    fillColor: '#5e6ad2',
    fillColorHover: '#7170ff',
  },
  Tabs: {
    tabTextColor: '#8a8f98',
    tabTextColorActive: '#1a1a2e',
    barColor: '#5e6ad2',
  },
}
