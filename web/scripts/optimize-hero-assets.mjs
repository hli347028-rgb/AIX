/**
 * 首页/电影感大图压缩：PNG → WebP（max 1920 宽，q≈72）
 * 保留原 PNG 文件名旁生成 .webp；体积目标单张数百 KB。
 */
import sharp from 'sharp'
import { readdir, stat } from 'node:fs/promises'
import { join, extname, basename } from 'node:path'

const kb = (n) => (n / 1024).toFixed(0) + ' KB'
const dir = 'public/assets'
const MAX_W = 1920
const QUALITY = 72

const files = (await readdir(dir))
  .filter((f) => /\.(png|jpe?g)$/i.test(f))
  .filter((f) => {
    // 小 logo 已单独优化，跳过
    const skip = /^(aix-logo|aix-coin|aix-logo-sm|logo|language|favicon)/i
    return !skip.test(f)
  })

let beforeTotal = 0
let afterTotal = 0

for (const file of files) {
  const src = join(dir, file)
  const before = (await stat(src)).size
  beforeTotal += before
  const outName = basename(file, extname(file)) + '.webp'
  const out = join(dir, outName)

  const img = sharp(src)
  const meta = await img.metadata()
  let pipeline = sharp(src)
  if (meta.width && meta.width > MAX_W) {
    pipeline = pipeline.resize(MAX_W, null, { withoutEnlargement: true })
  }
  await pipeline.webp({ quality: QUALITY, effort: 5 }).toFile(out)
  const after = (await stat(out)).size
  afterTotal += after
  const cut = (100 - (after / before) * 100).toFixed(1)
  console.log(`${file} -> ${outName}  ${kb(before)} -> ${kb(after)}  (-${cut}%)`)
}

console.log(`TOTAL source ${kb(beforeTotal)} -> webp ${kb(afterTotal)}`)
