-- name: GetProfile :one
SELECT * FROM profiles WHERE user_id = sqlc.arg(user_id);

-- name: UpsertProfile :exec
INSERT INTO profiles (user_id, first_name, mode, area_text, lat, lng, relocating, budget_min, budget_max, currency, move_earliest, move_latest, bio, contact_handle, duration_months, gender, gender_pref, occupation, searching, contact_self_confirmed)
VALUES (sqlc.arg(user_id), sqlc.arg(first_name), sqlc.arg(mode), sqlc.arg(area_text), sqlc.arg(lat), sqlc.arg(lng), sqlc.arg(relocating), sqlc.arg(budget_min), sqlc.arg(budget_max), sqlc.arg(currency), sqlc.arg(move_earliest), sqlc.arg(move_latest), sqlc.arg(bio), sqlc.arg(contact_handle), sqlc.arg(duration_months), sqlc.arg(gender), sqlc.arg(gender_pref), sqlc.arg(occupation), sqlc.arg(searching), sqlc.arg(contact_self_confirmed))
ON CONFLICT(user_id) DO UPDATE SET
  first_name = excluded.first_name,
  mode = excluded.mode,
  area_text = excluded.area_text,
  lat = excluded.lat,
  lng = excluded.lng,
  relocating = excluded.relocating,
  budget_min = excluded.budget_min,
  budget_max = excluded.budget_max,
  currency = excluded.currency,
  move_earliest = excluded.move_earliest,
  move_latest = excluded.move_latest,
  bio = excluded.bio,
  contact_handle = excluded.contact_handle,
  duration_months = excluded.duration_months,
  gender = excluded.gender,
  gender_pref = excluded.gender_pref,
  occupation = excluded.occupation,
  searching = excluded.searching,
  contact_self_confirmed = excluded.contact_self_confirmed;
