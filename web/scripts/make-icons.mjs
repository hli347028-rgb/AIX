/**
 * 把 GenerateImage 生成的"纯黑底 + 发光图标"转成带真实 alpha 通道的小图。
 *
 * 为什么不用 mix-blend-mode: screen：
 *   screen 混色只在纯黑背景上才正确，一旦图标叠在有渐变的卡片上就会发灰。
 *   直接把亮度写进 alpha 通道，图标就能在任何背景上正确合成。
 *
 * 用法：node scripts/make-icons.mjs
 */
import sharp from 'sharp'
import { readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'

const ROOT = process.cwd()

// 图标 CSS 显示尺寸是 46px，144 ≈ 3x，retina 屏足够锐利。
// 之前用 192 属于过度采样：文件大一倍，肉眼却看不出差别。
const OUT_SIZE = 144

// 原始大图放在 design-src/（不在 public/ 下）。
// 关键：public/ 里的东西会被原样部署且可公开下载，
// 把 2.2MB 的中间产物放那里等于白送带宽，也会被后端一起打包带走。
const JOBS = [
  { src: 'design-src/icons/icon-payment.png', dest: 'public/static/output-icon.png' },
  { src: 'design-src/icons/icon-game.png', dest: 'public/static/module-icon.png' },
  { src: 'design-src/icons/icon-chain.png', dest: 'public/static/chart-icon.png' },
]

/** 以亮度作为 alpha：黑 -> 全透明，亮 -> 不透明 */
async function blackToAlpha(srcPath) {
  const img = sharp(srcPath).ensureAlpha()
  const { data, info } = await img.raw().toBuffer({ resolveWithObject: true })
  const { width, height, channels } = info

  for (let i = 0; i < data.length; i += channels) {
    const r = data[i]
    const g = data[i + 1]
    const b = data[i + 2]

    // 取最大通道作为亮度近似（对青蓝色发光最稳）
    const lum = Math.max(r, g, b)

    // 低于阈值的当作纯背景，彻底透明；避免边缘残留一圈灰雾
    const alpha = lum <= 8 ? 0 : Math.min(255, Math.round(lum * 1.08))
    data[i + 3] = alpha

    // 反预乘补偿：alpha 变低会让颜色显灰，这里把 RGB 提亮回来
    if (alpha > 0 && alpha < 255) {
      const k = 255 / alpha
      data[i] = Math.min(255, Math.round(r * k * 0.55 + r * 0.45))
      data[i + 1] = Math.min(255, Math.round(g * k * 0.55 + g * 0.45))
      data[i + 2] = Math.min(255, Math.round(b * k * 0.55 + b * 0.45))
    }
  }

  return sharp(data, { raw: { width, height, channels } })
}

for (const { src, dest } of JOBS) {
  const srcAbs = path.join(ROOT, src)
  const destAbs = path.join(ROOT, dest)

  const withAlpha = await blackToAlpha(srcAbs)

  // trim 掉四周透明边，让图标铺满画面（原图留白很多，显示时会显得很小）
  const buf = await withAlpha
    .png()
    .toBuffer()
    // threshold 要够高才能裁掉四周那层很淡的光晕。
    // 用 2 的话光晕也算"内容"，图标主体会被挤得很小、显得没精神。
    .then((b) => sharp(b).trim({ threshold: 34 }).toBuffer())

  const before = (await readFile(srcAbs)).length

  const out = await sharp(buf)
    .resize(OUT_SIZE, OUT_SIZE, {
      fit: 'contain',
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    })
    // 不能开 palette：调色板会把 alpha 量化成很少几级，
    // 发光的柔和过渡会变成一圈方块状硬边。
    .png({ compressionLevel: 9, effort: 10 })
    .toBuffer()

  await writeFile(destAbs, out)
  console.log(
    `${path.basename(dest).padEnd(20)} ${(before / 1024).toFixed(0)}KB -> ${(out.length / 1024).toFixed(1)}KB`
  )
}
