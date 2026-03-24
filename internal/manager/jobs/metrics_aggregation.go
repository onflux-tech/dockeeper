package jobs

import (
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// ── Server metrics hourly ───────────────────────────────────────────────────

// AggregateServerMetricsHourly rolls up the previous full hour of raw server
// metrics into server_metrics_hourly.
func AggregateServerMetricsHourly(app core.App) {
	now := time.Now().UTC()
	end := now.Truncate(time.Hour)
	start := end.Add(-time.Hour)

	slog.Info("job: server metrics hourly aggregation", "start", start, "end", end)

	type aggRow struct {
		Server     string  `db:"server"`
		AvgCPU     float64 `db:"avg_cpu"`
		AvgMem     float64 `db:"avg_mem"`
		AvgMemUsed float64 `db:"avg_mem_used"`
		AvgMemTot  float64 `db:"avg_mem_total"`
		AvgDiskU   float64 `db:"avg_disk_used"`
		AvgDiskT   float64 `db:"avg_disk_total"`
		SumNetRx   float64 `db:"sum_net_rx"`
		SumNetTx   float64 `db:"sum_net_tx"`
		AvgLoad1   float64 `db:"avg_load_1m"`
		AvgLoad5   float64 `db:"avg_load_5m"`
		AvgLoad15  float64 `db:"avg_load_15m"`
		MinCPU     float64 `db:"min_cpu"`
		MaxCPU     float64 `db:"max_cpu"`
		MinMem     float64 `db:"min_mem"`
		MaxMem     float64 `db:"max_mem"`
		Samples    int     `db:"samples"`
	}

	var rows []aggRow
	err := app.DB().NewQuery(`
		SELECT server,
			AVG(cpu_percent) AS avg_cpu,
			AVG(mem_percent) AS avg_mem,
			AVG(mem_used)    AS avg_mem_used,
			AVG(mem_total)   AS avg_mem_total,
			AVG(disk_used)   AS avg_disk_used,
			AVG(disk_total)  AS avg_disk_total,
			SUM(net_rx)      AS sum_net_rx,
			SUM(net_tx)      AS sum_net_tx,
			AVG(load_1m)     AS avg_load_1m,
			AVG(load_5m)     AS avg_load_5m,
			AVG(load_15m)    AS avg_load_15m,
			MIN(cpu_percent) AS min_cpu,
			MAX(cpu_percent) AS max_cpu,
			MIN(mem_percent) AS min_mem,
			MAX(mem_percent) AS max_mem,
			COUNT(*)         AS samples
		FROM server_metrics_raw
		WHERE timestamp >= {:start} AND timestamp < {:end}
		GROUP BY server
	`).Bind(dbx.Params{
		"start": start.Format("2006-01-02 15:04:05.000Z"),
		"end":   end.Format("2006-01-02 15:04:05.000Z"),
	}).All(&rows)
	if err != nil {
		slog.Error("job: server hourly aggregation query failed", "error", err)
		return
	}

	col, err := app.FindCachedCollectionByNameOrId(constants.CollServerMetricsHourly)
	if err != nil {
		slog.Error("job: find server_metrics_hourly collection failed", "error", err)
		return
	}

	for _, r := range rows {
		// Compute p95 for cpu_percent and mem_percent.
		p95CPU := queryP95(app, "server_metrics_raw", "cpu_percent", "server", r.Server, start, end)
		p95Mem := queryP95(app, "server_metrics_raw", "mem_percent", "server", r.Server, start, end)

		stats := map[string]float64{
			"min_cpu_percent": r.MinCPU,
			"max_cpu_percent": r.MaxCPU,
			"p95_cpu_percent": p95CPU,
			"min_mem_percent": r.MinMem,
			"max_mem_percent": r.MaxMem,
			"p95_mem_percent": p95Mem,
		}
		statsJSON, _ := json.Marshal(stats)

		// Check if record already exists (idempotent).
		existing := findAggRecord(app, constants.CollServerMetricsHourly, "server", r.Server, start)
		rec := existing
		if rec == nil {
			rec = core.NewRecord(col)
			rec.Set("server", r.Server)
			rec.Set("timestamp", start)
		}

		rec.Set("cpu_percent", r.AvgCPU)
		rec.Set("mem_percent", r.AvgMem)
		rec.Set("mem_used", r.AvgMemUsed)
		rec.Set("mem_total", r.AvgMemTot)
		rec.Set("disk_used", r.AvgDiskU)
		rec.Set("disk_total", r.AvgDiskT)
		rec.Set("net_rx", r.SumNetRx)
		rec.Set("net_tx", r.SumNetTx)
		rec.Set("load_1m", r.AvgLoad1)
		rec.Set("load_5m", r.AvgLoad5)
		rec.Set("load_15m", r.AvgLoad15)
		rec.Set("samples", r.Samples)
		rec.Set("stats", string(statsJSON))

		if err := app.Save(rec); err != nil {
			slog.Error("job: save server hourly metric failed", "server", r.Server, "error", err)
		}
	}

	slog.Info("job: server metrics hourly aggregation done", "servers", len(rows))
}

// ── Server metrics daily ────────────────────────────────────────────────────

// AggregateServerMetricsDaily rolls up the previous full day of hourly server
// metrics into server_metrics_daily.
func AggregateServerMetricsDaily(app core.App) {
	now := time.Now().UTC()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -1)

	slog.Info("job: server metrics daily aggregation", "start", start, "end", end)

	type aggRow struct {
		Server     string  `db:"server"`
		AvgCPU     float64 `db:"avg_cpu"`
		AvgMem     float64 `db:"avg_mem"`
		AvgMemUsed float64 `db:"avg_mem_used"`
		AvgMemTot  float64 `db:"avg_mem_total"`
		AvgDiskU   float64 `db:"avg_disk_used"`
		AvgDiskT   float64 `db:"avg_disk_total"`
		SumNetRx   float64 `db:"sum_net_rx"`
		SumNetTx   float64 `db:"sum_net_tx"`
		AvgLoad1   float64 `db:"avg_load_1m"`
		AvgLoad5   float64 `db:"avg_load_5m"`
		AvgLoad15  float64 `db:"avg_load_15m"`
		MinCPU     float64 `db:"min_cpu"`
		MaxCPU     float64 `db:"max_cpu"`
		MinMem     float64 `db:"min_mem"`
		MaxMem     float64 `db:"max_mem"`
		TotSamples int     `db:"total_samples"`
	}

	var rows []aggRow
	err := app.DB().NewQuery(`
		SELECT server,
			AVG(cpu_percent) AS avg_cpu,
			AVG(mem_percent) AS avg_mem,
			AVG(mem_used)    AS avg_mem_used,
			AVG(mem_total)   AS avg_mem_total,
			AVG(disk_used)   AS avg_disk_used,
			AVG(disk_total)  AS avg_disk_total,
			SUM(net_rx)      AS sum_net_rx,
			SUM(net_tx)      AS sum_net_tx,
			AVG(load_1m)     AS avg_load_1m,
			AVG(load_5m)     AS avg_load_5m,
			AVG(load_15m)    AS avg_load_15m,
			MIN(cpu_percent) AS min_cpu,
			MAX(cpu_percent) AS max_cpu,
			MIN(mem_percent) AS min_mem,
			MAX(mem_percent) AS max_mem,
			SUM(samples)     AS total_samples
		FROM server_metrics_hourly
		WHERE timestamp >= {:start} AND timestamp < {:end}
		GROUP BY server
	`).Bind(dbx.Params{
		"start": start.Format("2006-01-02 15:04:05.000Z"),
		"end":   end.Format("2006-01-02 15:04:05.000Z"),
	}).All(&rows)
	if err != nil {
		slog.Error("job: server daily aggregation query failed", "error", err)
		return
	}

	col, err := app.FindCachedCollectionByNameOrId(constants.CollServerMetricsDaily)
	if err != nil {
		slog.Error("job: find server_metrics_daily collection failed", "error", err)
		return
	}

	for _, r := range rows {
		p95CPU := queryP95(app, "server_metrics_hourly", "cpu_percent", "server", r.Server, start, end)
		p95Mem := queryP95(app, "server_metrics_hourly", "mem_percent", "server", r.Server, start, end)

		stats := map[string]float64{
			"min_cpu_percent": r.MinCPU,
			"max_cpu_percent": r.MaxCPU,
			"p95_cpu_percent": p95CPU,
			"min_mem_percent": r.MinMem,
			"max_mem_percent": r.MaxMem,
			"p95_mem_percent": p95Mem,
		}
		statsJSON, _ := json.Marshal(stats)

		existing := findAggRecord(app, constants.CollServerMetricsDaily, "server", r.Server, start)
		rec := existing
		if rec == nil {
			rec = core.NewRecord(col)
			rec.Set("server", r.Server)
			rec.Set("timestamp", start)
		}

		rec.Set("cpu_percent", r.AvgCPU)
		rec.Set("mem_percent", r.AvgMem)
		rec.Set("mem_used", r.AvgMemUsed)
		rec.Set("mem_total", r.AvgMemTot)
		rec.Set("disk_used", r.AvgDiskU)
		rec.Set("disk_total", r.AvgDiskT)
		rec.Set("net_rx", r.SumNetRx)
		rec.Set("net_tx", r.SumNetTx)
		rec.Set("load_1m", r.AvgLoad1)
		rec.Set("load_5m", r.AvgLoad5)
		rec.Set("load_15m", r.AvgLoad15)
		rec.Set("samples", r.TotSamples)
		rec.Set("stats", string(statsJSON))

		if err := app.Save(rec); err != nil {
			slog.Error("job: save server daily metric failed", "server", r.Server, "error", err)
		}
	}

	slog.Info("job: server metrics daily aggregation done", "servers", len(rows))
}

// ── Container metrics hourly ────────────────────────────────────────────────

// AggregateContainerMetricsHourly rolls up the previous full hour of raw
// container metrics into container_metrics_hourly.
func AggregateContainerMetricsHourly(app core.App) {
	now := time.Now().UTC()
	end := now.Truncate(time.Hour)
	start := end.Add(-time.Hour)

	slog.Info("job: container metrics hourly aggregation", "start", start, "end", end)

	type aggRow struct {
		Container  string  `db:"container"`
		Server     string  `db:"server"`
		AvgCPU     float64 `db:"avg_cpu"`
		AvgMemUsed float64 `db:"avg_mem_used"`
		AvgMemLim  float64 `db:"avg_mem_limit"`
		SumNetRx   float64 `db:"sum_net_rx"`
		SumNetTx   float64 `db:"sum_net_tx"`
		SumBlkR    float64 `db:"sum_block_read"`
		SumBlkW    float64 `db:"sum_block_write"`
		AvgPIDs    float64 `db:"avg_pids"`
		MinCPU     float64 `db:"min_cpu"`
		MaxCPU     float64 `db:"max_cpu"`
		Samples    int     `db:"samples"`
	}

	var rows []aggRow
	err := app.DB().NewQuery(`
		SELECT container, server,
			AVG(cpu_percent)  AS avg_cpu,
			AVG(mem_used)     AS avg_mem_used,
			AVG(mem_limit)    AS avg_mem_limit,
			SUM(net_rx)       AS sum_net_rx,
			SUM(net_tx)       AS sum_net_tx,
			SUM(block_read)   AS sum_block_read,
			SUM(block_write)  AS sum_block_write,
			AVG(pids)         AS avg_pids,
			MIN(cpu_percent)  AS min_cpu,
			MAX(cpu_percent)  AS max_cpu,
			COUNT(*)          AS samples
		FROM container_metrics_raw
		WHERE timestamp >= {:start} AND timestamp < {:end}
		GROUP BY container, server
	`).Bind(dbx.Params{
		"start": start.Format("2006-01-02 15:04:05.000Z"),
		"end":   end.Format("2006-01-02 15:04:05.000Z"),
	}).All(&rows)
	if err != nil {
		slog.Error("job: container hourly aggregation query failed", "error", err)
		return
	}

	col, err := app.FindCachedCollectionByNameOrId(constants.CollContainerMetricsHourly)
	if err != nil {
		slog.Error("job: find container_metrics_hourly collection failed", "error", err)
		return
	}

	for _, r := range rows {
		p95CPU := queryP95(app, "container_metrics_raw", "cpu_percent", "container", r.Container, start, end)

		stats := map[string]float64{
			"min_cpu_percent": r.MinCPU,
			"max_cpu_percent": r.MaxCPU,
			"p95_cpu_percent": p95CPU,
		}
		statsJSON, _ := json.Marshal(stats)

		existing := findContainerAggRecord(app, constants.CollContainerMetricsHourly, r.Container, start)
		rec := existing
		if rec == nil {
			rec = core.NewRecord(col)
			rec.Set("container", r.Container)
			rec.Set("server", r.Server)
			rec.Set("timestamp", start)
		}

		rec.Set("cpu_percent", r.AvgCPU)
		rec.Set("mem_used", r.AvgMemUsed)
		rec.Set("mem_limit", r.AvgMemLim)
		rec.Set("net_rx", r.SumNetRx)
		rec.Set("net_tx", r.SumNetTx)
		rec.Set("block_read", r.SumBlkR)
		rec.Set("block_write", r.SumBlkW)
		rec.Set("pids", int(math.Round(r.AvgPIDs)))
		rec.Set("samples", r.Samples)
		rec.Set("stats", string(statsJSON))

		if err := app.Save(rec); err != nil {
			slog.Error("job: save container hourly metric failed", "container", r.Container, "error", err)
		}
	}

	slog.Info("job: container metrics hourly aggregation done", "containers", len(rows))
}

// ── Container metrics daily ─────────────────────────────────────────────────

// AggregateContainerMetricsDaily rolls up the previous full day of hourly
// container metrics into container_metrics_daily.
func AggregateContainerMetricsDaily(app core.App) {
	now := time.Now().UTC()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -1)

	slog.Info("job: container metrics daily aggregation", "start", start, "end", end)

	type aggRow struct {
		Container  string  `db:"container"`
		Server     string  `db:"server"`
		AvgCPU     float64 `db:"avg_cpu"`
		AvgMemUsed float64 `db:"avg_mem_used"`
		AvgMemLim  float64 `db:"avg_mem_limit"`
		SumNetRx   float64 `db:"sum_net_rx"`
		SumNetTx   float64 `db:"sum_net_tx"`
		SumBlkR    float64 `db:"sum_block_read"`
		SumBlkW    float64 `db:"sum_block_write"`
		AvgPIDs    float64 `db:"avg_pids"`
		MinCPU     float64 `db:"min_cpu"`
		MaxCPU     float64 `db:"max_cpu"`
		TotSamples int     `db:"total_samples"`
	}

	var rows []aggRow
	err := app.DB().NewQuery(`
		SELECT container, server,
			AVG(cpu_percent)  AS avg_cpu,
			AVG(mem_used)     AS avg_mem_used,
			AVG(mem_limit)    AS avg_mem_limit,
			SUM(net_rx)       AS sum_net_rx,
			SUM(net_tx)       AS sum_net_tx,
			SUM(block_read)   AS sum_block_read,
			SUM(block_write)  AS sum_block_write,
			AVG(pids)         AS avg_pids,
			MIN(cpu_percent)  AS min_cpu,
			MAX(cpu_percent)  AS max_cpu,
			SUM(samples)      AS total_samples
		FROM container_metrics_hourly
		WHERE timestamp >= {:start} AND timestamp < {:end}
		GROUP BY container, server
	`).Bind(dbx.Params{
		"start": start.Format("2006-01-02 15:04:05.000Z"),
		"end":   end.Format("2006-01-02 15:04:05.000Z"),
	}).All(&rows)
	if err != nil {
		slog.Error("job: container daily aggregation query failed", "error", err)
		return
	}

	col, err := app.FindCachedCollectionByNameOrId(constants.CollContainerMetricsDaily)
	if err != nil {
		slog.Error("job: find container_metrics_daily collection failed", "error", err)
		return
	}

	for _, r := range rows {
		p95CPU := queryP95(app, "container_metrics_hourly", "cpu_percent", "container", r.Container, start, end)

		stats := map[string]float64{
			"min_cpu_percent": r.MinCPU,
			"max_cpu_percent": r.MaxCPU,
			"p95_cpu_percent": p95CPU,
		}
		statsJSON, _ := json.Marshal(stats)

		existing := findContainerAggRecord(app, constants.CollContainerMetricsDaily, r.Container, start)
		rec := existing
		if rec == nil {
			rec = core.NewRecord(col)
			rec.Set("container", r.Container)
			rec.Set("server", r.Server)
			rec.Set("timestamp", start)
		}

		rec.Set("cpu_percent", r.AvgCPU)
		rec.Set("mem_used", r.AvgMemUsed)
		rec.Set("mem_limit", r.AvgMemLim)
		rec.Set("net_rx", r.SumNetRx)
		rec.Set("net_tx", r.SumNetTx)
		rec.Set("block_read", r.SumBlkR)
		rec.Set("block_write", r.SumBlkW)
		rec.Set("pids", int(math.Round(r.AvgPIDs)))
		rec.Set("samples", r.TotSamples)
		rec.Set("stats", string(statsJSON))

		if err := app.Save(rec); err != nil {
			slog.Error("job: save container daily metric failed", "container", r.Container, "error", err)
		}
	}

	slog.Info("job: container metrics daily aggregation done", "containers", len(rows))
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// queryP95 fetches sorted values of a metric column and returns the 95th
// percentile. It scopes by a key column (e.g. "server" or "container").
func queryP95(app core.App, table, column, keyCol, keyVal string, start, end time.Time) float64 {
	type valRow struct {
		Val float64 `db:"val"`
	}

	var vals []valRow
	//nolint:gosec // table/column names are internal constants, not user input
	sql := "SELECT " + column + " AS val FROM " + table +
		" WHERE " + keyCol + " = {:key} AND timestamp >= {:start} AND timestamp < {:end}" +
		" ORDER BY " + column + " ASC"

	err := app.DB().NewQuery(sql).Bind(dbx.Params{
		"key":   keyVal,
		"start": start.Format("2006-01-02 15:04:05.000Z"),
		"end":   end.Format("2006-01-02 15:04:05.000Z"),
	}).All(&vals)
	if err != nil || len(vals) == 0 {
		return 0
	}

	sorted := make([]float64, len(vals))
	for i, v := range vals {
		sorted[i] = v.Val
	}
	sort.Float64s(sorted)

	idx := int(float64(len(sorted)-1) * 0.95)
	return sorted[idx]
}

// findAggRecord looks for an existing aggregated record by key relation + timestamp.
func findAggRecord(app core.App, collection, keyField, keyValue string, ts time.Time) *core.Record {
	records, err := app.FindRecordsByFilter(
		collection,
		keyField+" = {:key} && timestamp = {:ts}",
		"", 1, 0,
		map[string]any{
			"key": keyValue,
			"ts":  ts.Format("2006-01-02 15:04:05.000Z"),
		},
	)
	if err != nil || len(records) == 0 {
		return nil
	}
	return records[0]
}

// findContainerAggRecord looks for an existing container aggregated record.
func findContainerAggRecord(app core.App, collection, containerID string, ts time.Time) *core.Record {
	return findAggRecord(app, collection, "container", containerID, ts)
}
