package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"backend/internal/data"

	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	host := flag.String("host", "127.0.0.1", "MySQL host (local only)")
	port := flag.Int("port", 3306, "MySQL port")
	user := flag.String("user", "root", "MySQL user")
	pass := flag.String("pass", "root", "MySQL password")
	dbname := flag.String("db", "aix", "MySQL database")
	address := flag.String("address", "0x588A0B6fEf36a03621D42e92a26918327D08Cc5E", "target user address")
	amount := flag.String("amount", "350", "clawback amount")
	dryRun := flag.Bool("dry-run", false, "simulate only, do not write")
	flag.Parse()

	h := strings.TrimSpace(*host)
	if h != "127.0.0.1" && !strings.EqualFold(h, "localhost") {
		fmt.Fprintln(os.Stderr, "refusing non-local host; this tool is local-test only")
		os.Exit(2)
	}

	target, err := decimal.NewFromString(*amount)
	if err != nil || !target.IsPositive() {
		fmt.Fprintln(os.Stderr, "invalid amount")
		os.Exit(2)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		*user, *pass, h, *port, *dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "db open:", err)
		os.Exit(1)
	}

	rep, err := data.RunMgmtClawback(db, data.ClawbackOptions{
		Address:                *address,
		TargetAmount:           target,
		TreatAllActiveAsReward: true,
		DryRun:                 *dryRun,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "clawback:", err)
		os.Exit(1)
	}

	mode := "APPLY"
	if *dryRun {
		mode = "DRY-RUN"
	}
	fmt.Printf("=== local clawback %s ===\n%s", mode, data.FormatClawbackReport(rep))
	if rep.Unrecovered.IsPositive() {
		fmt.Printf("UNRECOVERED=%s\n", rep.Unrecovered)
	}
}
