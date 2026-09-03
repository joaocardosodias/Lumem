INSERT INTO stations (id, code, name) VALUES
(1, 'VSO', 'Vila Sônia'),
(2, 'MBI', 'São Paulo - Morumbi'),
(3, 'BUT', 'Butantã'),
(4, 'PIN', 'Pinheiros'),
(5, 'FAL', 'Faria Lima'),
(6, 'FRA', 'Fradique Coutinho'),
(7, 'OFR', 'Oscar Freire'),
(8, 'PAU', 'Paulista'),
(9, 'HIG', 'Higienópolis - Mackenzie'),
(10, 'REP', 'República'),
(11, 'LUZ', 'Luz');

WITH RECURSIVE
  doors_seq(n) AS (
    SELECT 1 UNION ALL SELECT n + 1 FROM doors_seq WHERE n < 24
  ),
  platforms(plt) AS (
    SELECT 'P1' UNION ALL SELECT 'P2'
  )
INSERT INTO doors (station_id, platform, door_number, sap_tag, status)
SELECT 
    s.id,
    p.plt,
    d.n,
    'SP04-6-PSD-' || s.code || '-' || p.plt || '-PP' || CASE WHEN d.n < 10 THEN '0' || d.n ELSE '' || d.n END,
    'active'
FROM stations s
CROSS JOIN platforms p
CROSS JOIN doors_seq d;
