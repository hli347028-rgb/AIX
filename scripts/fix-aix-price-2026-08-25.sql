UPDATE aix_prices SET price = 0.104040000000000000, remark = 'daily +2%' WHERE effective_date = '2026-08-25';
UPDATE settings SET value = JSON_SET(value, '$.aix_price_initial', 0.10404) WHERE `key` = 'system_config';
SELECT effective_date, price, remark FROM aix_prices WHERE effective_date >= '2026-08-23' ORDER BY effective_date;
