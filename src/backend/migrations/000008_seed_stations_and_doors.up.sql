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
(11, 'LUZ', 'Luz')
ON CONFLICT (id) DO NOTHING;

WITH platforms(plt, offset_p) AS (
    SELECT 'P1', 0 UNION ALL SELECT 'P2', 24
),
doors_seq(n) AS (
    SELECT generate_series(1, 24)
)
INSERT INTO doors (id, station_id, platform, door_number, sap_tag, status)
SELECT 
    (s.id - 1) * 48 + p.offset_p + d.n,
    s.id,
    p.plt,
    d.n,
    'SP04-6-PSD-' || s.code || '-' || p.plt || '-PP' || LPAD(d.n::text, 2, '0'),
    'active'
FROM stations s
CROSS JOIN platforms p
CROSS JOIN doors_seq d
ON CONFLICT (id) DO NOTHING;
