export function formatAmount4 (value) {
    if (value == null || value === '') return '0.0000'
    const n = parseFloat(value)
    if (!Number.isFinite(n)) return String(value)
    return n.toFixed(4)
}

export function formatStat4 (value) {
    return formatAmount4(value)
}
