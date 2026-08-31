SET NAMES utf8mb4;

UPDATE announcements
SET
  title = '关于 WIN-AIX 划转及 AIX 系统功能开放的公告',
  content = CONCAT(
    '<p>WIN-AIX 划转报单、AIX-USDT 提现与引擎购买、AIX 充值提现及认购功能现已开放。</p>',
    '<p>请查看完整公告图片了解功能开放时间、划转要求及操作说明。</p>',
    '<p><img src="https://hebbkx1anhila5yf.public.blob.vercel-storage.com/image-w1OZAPxUschUv0hFqbc8iFMuaUES0Q.png" alt="公告图片" style="max-width:100%;height:auto;display:block;margin:12px 0;" /></p>'
  ),
  status = 1,
  created_time = '2026-08-30 22:00:00.000',
  updated_time = NOW(3)
WHERE id = 1;

SELECT id, title, status, created_time, CHAR_LENGTH(title) AS title_len FROM announcements WHERE id = 1;
