package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"backend/internal/biz"
	"backend/internal/data"

	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	host := flag.String("host", "127.0.0.1", "MySQL host")
	port := flag.Int("port", 3306, "MySQL port")
	user := flag.String("user", "root", "MySQL user")
	pass := flag.String("pass", "root", "MySQL password")
	dbname := flag.String("db", "aix_prod_dry", "MySQL database")
	dryRun := flag.Bool("dry-run", true, "simulate only (default true)")
	allowApply := flag.Bool("allow-apply", false, "required together with -dry-run=false to write")
	outPath := flag.String("out", "", "optional report file path")
	flag.Parse()

	if !*dryRun && !*allowApply {
		fmt.Fprintln(os.Stderr, "refusing apply without -allow-apply")
		os.Exit(2)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		*user, *pass, *host, *port, *dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "db open:", err)
		os.Exit(1)
	}

	type dueRow struct {
		UserID  int64
		Address string
		Due     decimal.Decimal
	}
	var dues []dueRow
	err = db.Raw(`
SELECT u.id AS user_id, u.address, COALESCE(SUM(m.total_amount),0) AS due
FROM mgmt_rewards m
JOIN orders o ON o.id = m.source_order_id
JOIN users u ON u.id = m.user_id
WHERE o.fund_source = ?
GROUP BY u.id, u.address
HAVING due > 0
ORDER BY due DESC`, biz.PayFromReward).Scan(&dues).Error
	if err != nil {
		fmt.Fprintln(os.Stderr, "load dues:", err)
		os.Exit(1)
	}

	var b strings.Builder
	mode := "DRY-RUN"
	if !*dryRun {
		mode = "APPLY"
	}
	fmt.Fprintf(&b, "=== batch clawback %s db=%s users=%d ===\n", mode, *dbname, len(dues))

	totalDue := decimal.Zero
	totalUnrec := decimal.Zero
	totalPtsShort := decimal.Zero
	unrecUsers := 0
	ptsShortUsers := 0

	for _, d := range dues {
		rep, err := data.RunMgmtClawback(db, data.ClawbackOptions{
			Address:                d.Address,
			TargetAmount:           d.Due,
			TreatAllActiveAsReward: false,
			DryRun:                 *dryRun,
		})
		if err != nil {
			fmt.Fprintf(&b, "ERROR user=%s due=%s: %v\n", d.Address, d.Due, err)
			continue
		}
		totalDue = totalDue.Add(rep.TargetAmount)
		fmt.Fprint(&b, data.FormatClawbackReport(rep))
		if rep.Unrecovered.IsPositive() {
			unrecUsers++
			totalUnrec = totalUnrec.Add(rep.Unrecovered)
			fmt.Fprintf(&b, "  !! MGMT_UNRECOVERED address=%s amount=%s\n", rep.Address, rep.Unrecovered)
		}
		if rep.PointsShortfall.IsPositive() {
			ptsShortUsers++
			totalPtsShort = totalPtsShort.Add(rep.PointsShortfall)
			fmt.Fprintf(&b, "  !! POINTS_SHORT address=%s amount=%s\n", rep.Address, rep.PointsShortfall)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "=== SUMMARY ===\n")
	fmt.Fprintf(&b, "total_due=%s unrecovered_users=%d unrecovered_sum=%s\n", totalDue, unrecUsers, totalUnrec)
	fmt.Fprintf(&b, "points_short_users=%d points_short_sum=%s\n", ptsShortUsers, totalPtsShort)

	text := b.String()
	fmt.Print(text)
	if *outPath != "" {
		if err := os.WriteFile(*outPath, []byte(text), 0644); err != nil {
			fmt.Fprintln(os.Stderr, "write out:", err)
			os.Exit(1)
		}
		fmt.Println("wrote", *outPath)
	}
}
