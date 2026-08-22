export interface UserLanguageOption {
  code: string
  name: string
}

/** 用户端可选语言（已有语言请勿重复添加） */
export const userLanguageOptions: UserLanguageOption[] = [
  { code: 'zh', name: '简体中文' },
  { code: 'zh-tw', name: '繁體中文' },
  { code: 'en', name: 'English' },
  { code: 'th', name: 'ไทย' },
  { code: 'vi', name: 'Tiếng Việt' },
  { code: 'ko', name: '한국어' },
  { code: 'id', name: 'Bahasa Indonesia' },
  { code: 'ja', name: '日本語' },
]
