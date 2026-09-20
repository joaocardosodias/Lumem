WITH ranked_open_inspections AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY station_code, platform, door_number
               ORDER BY CASE WHEN status = 'in_progress' THEN 0 ELSE 1 END,
                        created_at,
                        id
           ) AS position
    FROM inspections
    WHERE status IN ('pending', 'in_progress')
)
UPDATE inspections AS inspection
SET status = 'canceled',
    notes = CASE
        WHEN NULLIF(BTRIM(inspection.notes), '') IS NULL
            THEN 'Ordem duplicada encerrada automaticamente durante a reconciliacao.'
        ELSE inspection.notes || E'\nOrdem duplicada encerrada automaticamente durante a reconciliacao.'
    END,
    completed_at = NULL,
    updated_at = CURRENT_TIMESTAMP
FROM ranked_open_inspections AS ranked
WHERE inspection.id = ranked.id
  AND ranked.position > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_inspections_one_open_per_door
    ON inspections (station_code, platform, door_number)
    WHERE status IN ('pending', 'in_progress');
