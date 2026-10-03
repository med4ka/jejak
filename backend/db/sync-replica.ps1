# Manual replica sync: DEV TESTING ONLY, dijalankan manual sesuai kebutuhan.
#
# Menyalin isi creators + urls + click_events dari primary (jejak) ke
# replica (jejak_replica) dalam 1 perintah, supaya tidak tulis query manual
# tiap kali mau testing.
#
# Ini TIDAK mengganti prinsip "replication lag itu sengaja untuk belajar"
# (ARCHITECTURE.md §6): replikasi Postgres asli sync OTOMATIS; di sini ANDA
# yang memutuskan kapan sync terjadi. Jangan pernah dijalankan otomatis
# (cron/service/scheduler): kalau otomatis, nilai belajarnya hilang.
#
# Aman diulang (idempotent): replica tidak menyimpan data unik (app tidak
# pernah tulis ke replica), jadi DELETE + salin ulang tidak menghilangkan
# apa-apa. Sequence diperbaiki setelah salin supaya INSERT berikutnya benar.

$pgBin = "C:\Program Files\PostgreSQL\17\bin"
$pgUser = "postgres"
$pgHost = "localhost"
$primary = "jejak"
$replica = "jejak_replica"

Write-Host "[1/3] Mengosongkan replica (children dulu supaya FK aman)..."
& "$pgBin\psql.exe" -U $pgUser -h $pgHost -d $replica -c "DELETE FROM click_events; DELETE FROM urls; DELETE FROM creators;"
if ($LASTEXITCODE -ne 0) { Write-Error "Gagal mengosongkan replica"; exit 1 }

Write-Host "[2/3] Menyalin data primary -> replica..."
& "$pgBin\pg_dump.exe" -U $pgUser -h $pgHost --data-only --column-inserts -t creators -t urls -t click_events $primary |
    & "$pgBin\psql.exe" -U $pgUser -h $pgHost -d $replica -q -v ON_ERROR_STOP=1
if ($LASTEXITCODE -ne 0) { Write-Error "Gagal menyalin data"; exit 1 }

Write-Host "[3/3] Memperbaiki sequence..."
& "$pgBin\psql.exe" -U $pgUser -h $pgHost -d $replica -c "SELECT setval('creators_id_seq', COALESCE((SELECT max(id) FROM creators),0)+1, false); SELECT setval('urls_id_seq', COALESCE((SELECT max(id) FROM urls),0)+1, false); SELECT setval('click_events_id_seq', COALESCE((SELECT max(id) FROM click_events),0)+1, false);"
if ($LASTEXITCODE -ne 0) { Write-Error "Gagal memperbaiki sequence"; exit 1 }

Write-Host "Sync selesai."
