type Messages = Record<string, any>

export function mergeMessages<T extends Messages, U extends Messages>(base: T, override: U): T & U {
  const result: Messages = { ...base }
  for (const [key, value] of Object.entries(override)) {
    const baseValue = result[key]
    result[key] = value && baseValue && typeof value === 'object' && typeof baseValue === 'object'
      && !Array.isArray(value) && !Array.isArray(baseValue)
      ? mergeMessages(baseValue, value)
      : value
  }
  return result as T & U
}
