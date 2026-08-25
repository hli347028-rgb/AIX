INSERT INTO aix_prices (price, effective_date, remark)
VALUES (0.102000000000000000, '2026-08-25', 'daily +2%')
ON DUPLICATE KEY UPDATE price = VALUES(price), remark = VALUES(remark);
SELECT effective_date, price, remark FROM aix_prices ORDER BY effective_date DESC LIMIT 5;
