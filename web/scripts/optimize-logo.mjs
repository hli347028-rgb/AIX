/**
 * Logo 体积优化（不改变设计，仅重采样 + 压缩）
 *
 * 背景：logo.png 是 1024x1024 的未压缩 PNG。
 *   public/assets/logo.png  2032 KB
 *   public/favicon.png      2032 KB
 *   src/assets/images/logo.png 1605 KB
 * 合计约 5.5MB，而它在 Header 里只显示 ~40px、首页 hero 最大 ~250px。
 * 对一个移动端 DApp 来说这是最昂贵的一处浪费（favicon 还会阻塞首屏）。
 *
 * 处理策略：
 *   - hero/header 用 512x512：即使 3x DPR 下显示 250px 也够（750px > 512 略有折损，
 *     但 logo 是发光实心图形，512 足够；再大收益极小、体积翻倍）
 *   - favicon 用 256x256：apple-touch-icon 推荐 180px，256 有余量
 *   - palette 量化 + 最高压缩等级，保留 alpha
 *
 * 原图已备份至 design-src/logo-original.png。
 */
import sharp from 'sharp'
import { copyFile, mkdir, stat } from 'node:fs/promises'

const kb = (n) => (n / 1024).toFixed(0) + ' KB'

/* 这个仓库里同一个 logo 存了 5 份副本（历史遗留），逐个处理。
   src/assets/favicon.png 才是真正被打包成 dist favicon 的那一份。 */
const targets = [
  { src: 'public/assets/logo.png', out: 'public/assets/logo.png', size: 512 },
  { src: 'public/assets/aix-logo.png', out: 'public/assets/aix-logo.png', size: 512 },
  { src: 'public/favicon.png', out: 'public/favicon.png', size: 256 },
  { src: 'src/assets/images/logo.png', out: 'src/assets/images/logo.png', size: 512 },
  { src: 'src/assets/favicon.png', out: 'src/assets/favicon.png', size: 256 }
]

await mkdir('design-src', { recursive: true })
await copyFile('public/assets/logo.png', 'design-src/logo-original.png')

for (const t of targets) {
  const before = (await stat(t.src)).size

  // 先解码到内存，避免读写同一路径导致的竞态
  const buf = await sharp(t.src)
    .resize(t.size, t.size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
    .png({ compressionLevel: 9, palette: true, quality: 90, effort: 10 })
    .toBuffer()

  await sharp(buf).toFile(t.out)
  const after = (await stat(t.out)).size
  const cut = (100 - (after / before) * 100).toFixed(1)
  console.log(`${t.out}  ${kb(before)} -> ${kb(after)}  (-${cut}%)`)
}
