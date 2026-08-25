import fs from 'fs'
import path from 'path'
import { PNG } from 'pngjs'

const files = [
  'public/assets/logo.png',
  'public/favicon.png',
  'src/assets/images/logo.png',
  'src/assets/favicon.png',
]

const root = path.resolve(import.meta.dirname, '..')

async function loadImage(inputPath) {
  const buf = fs.readFileSync(inputPath)
  const sig = buf.slice(0, 4).toString('hex')
  if (sig.startsWith('89504e47')) {
    return PNG.sync.read(buf)
  }
  const { createRequire } = await import('module')
  const require = createRequire(import.meta.url)
  let jpeg
  try {
    jpeg = require('jpeg-js')
  } catch {
    throw new Error(`Unsupported image format (${sig}) and jpeg-js is unavailable`)
  }
  const decoded = jpeg.decode(buf, { useTArray: true })
  const png = new PNG({ width: decoded.width, height: decoded.height })
  png.data = Buffer.from(decoded.data)
  return png
}

async function processPng(inputPath, outputPath) {
  const png = await loadImage(inputPath)
  const { width, height, data } = png
  const cx = (width - 1) / 2
  const cy = (height - 1) / 2
  const radius = Math.min(width, height) * 0.495
  const radiusSq = radius * radius

  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = (width * y + x) << 2
      const r = data[idx]
      const g = data[idx + 1]
      const b = data[idx + 2]
      const dx = x - cx
      const dy = y - cy
      const distSq = dx * dx + dy * dy
      const isDark = r < 28 && g < 28 && b < 28
      const outsideCircle = distSq > radiusSq

      if (outsideCircle || isDark) {
        data[idx + 3] = 0
      }
    }
  }

  fs.mkdirSync(path.dirname(outputPath), { recursive: true })
  fs.writeFileSync(outputPath, PNG.sync.write(png))
  console.log('ok', outputPath)
}

const source = process.argv[2]
  ? path.resolve(process.argv[2])
  : path.join(root, 'public/assets/logo.png')

for (const rel of files) {
  await processPng(source, path.join(root, rel))
}
