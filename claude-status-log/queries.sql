-- This uses MAX(id) per session_id to grab the most recent row for each session. You can pull specific fields from the JSON too, e.g.:
SELECT
      session_id,
      recorded_at,
      raw ->> '$.model.display_name' AS model,
      raw ->> '$.cost.total_cost_usd' AS cost
  FROM status
  WHERE id IN (
      SELECT MAX(id) FROM status GROUP BY session_id
  )
  ORDER BY recorded_at DESC;


-- grab latest
SELECT session_id, recorded_at, json(raw)
  FROM status
  WHERE id IN (
      SELECT MAX(id) FROM status GROUP BY session_id
  )
  ORDER BY recorded_at DESC;



-- select token usage
SELECT
    session_id,
    recorded_at,
    raw ->> '$.model.display_name' AS model,
    raw ->> '$.cost.total_cost_usd' AS cost,
    raw ->> '$.context_window.total_input_tokens' AS input_tokens,
    raw ->> '$.context_window.total_output_tokens' AS output_tokens
FROM status
WHERE id IN (
    SELECT MAX(id) FROM status GROUP BY session_id
)
ORDER BY recorded_at DESC;