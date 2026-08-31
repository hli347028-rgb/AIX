package data

import (
	"context"
	"fmt"

	"backend/internal/biz"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// refreshPerformance recalculates cached performance from cumulative order
// principal (active + exited). Performance only increases on new subscriptions;
// downline orders completing release do not reduce team/small-area figures.
//
// seedMode:
//   - "" / full: refresh every user
//   - "ancestors": refresh parents of each seed (not the seed itself); used after subscribe
//   - "seeds": refresh each seed and its ancestors; used after changing inviter
func refreshPerformance(db *gorm.DB, seedMode string, seedIDs ...int64) error {
	type userNode struct {
		ID              int64
		InviterID       *int64
		MgmtLevelLocked bool
	}
	type principalRow struct {
		UserID int64
		Total  decimal.Decimal
	}

	var users []userNode
	if err := db.Model(&UserPO{}).Select("id", "inviter_id", "mgmt_level_locked").Find(&users).Error; err != nil {
		return err
	}
	var principals []principalRow
	if err := db.Model(&OrderPO{}).
		Select("user_id, COALESCE(SUM(principal), 0) AS total").
		Where("status IN ?", []string{biz.OrderStatusActive, biz.OrderStatusExited}).
		Group("user_id").Scan(&principals).Error; err != nil {
		return err
	}

	children := make(map[int64][]int64, len(users))
	parents := make(map[int64]int64, len(users))
	userByID := make(map[int64]userNode, len(users))
	stake := make(map[int64]decimal.Decimal, len(users))
	for _, user := range users {
		userByID[user.ID] = user
		stake[user.ID] = decimal.Zero
		if user.InviterID != nil {
			children[*user.InviterID] = append(children[*user.InviterID], user.ID)
			parents[user.ID] = *user.InviterID
		}
	}
	for _, row := range principals {
		stake[row.UserID] = row.Total
	}

	memo := make(map[int64]decimal.Decimal, len(users))
	visiting := make(map[int64]bool, len(users))
	var subtree func(int64) (decimal.Decimal, error)
	subtree = func(id int64) (decimal.Decimal, error) {
		if value, ok := memo[id]; ok {
			return value, nil
		}
		if visiting[id] {
			return decimal.Zero, fmt.Errorf("invite relationship contains a cycle at user %d", id)
		}
		visiting[id] = true
		total := stake[id]
		for _, childID := range children[id] {
			childTotal, err := subtree(childID)
			if err != nil {
				return decimal.Zero, err
			}
			total = total.Add(childTotal)
		}
		visiting[id] = false
		memo[id] = total
		return total, nil
	}

	targets := users
	if seedMode == "ancestors" || seedMode == "seeds" {
		targets = make([]userNode, 0)
		seen := map[int64]bool{}
		for _, seedID := range seedIDs {
			if seedID <= 0 {
				continue
			}
			currentID := seedID
			if seedMode == "ancestors" {
				parentID, ok := parents[currentID]
				if !ok {
					continue
				}
				currentID = parentID
			}
			pathSeen := map[int64]bool{}
			for currentID > 0 {
				if pathSeen[currentID] {
					return fmt.Errorf("invite relationship contains a cycle at user %d", currentID)
				}
				pathSeen[currentID] = true
				if !seen[currentID] {
					node, ok := userByID[currentID]
					if !ok {
						break
					}
					seen[currentID] = true
					targets = append(targets, node)
				}
				parentID, ok := parents[currentID]
				if !ok {
					break
				}
				currentID = parentID
			}
		}
	}

	for _, user := range targets {
		branches := make([]decimal.Decimal, 0, len(children[user.ID]))
		for _, childID := range children[user.ID] {
			value, err := subtree(childID)
			if err != nil {
				return err
			}
			branches = append(branches, value)
		}
		large, small, team := biz.CalcAreaPerformance(branches)
		// 晋级：大区与小区须同时达到同一门槛；取两者可晋级等级的较低者
		level := biz.MgmtLevelByAreas(large, small)
		updates := map[string]interface{}{
			"large_area_perf": large,
			"small_area_perf": small,
			"team_perf":       team,
		}
		if !user.MgmtLevelLocked {
			updates["mgmt_level"] = level
		}
		if err := db.Model(&UserPO{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func refreshAllPerformance(db *gorm.DB) error {
	return refreshPerformance(db, "")
}

func refreshAncestorPerformance(db *gorm.DB, sourceUserID int64) error {
	return refreshPerformance(db, "ancestors", sourceUserID)
}

func refreshSeedAndAncestors(db *gorm.DB, seedIDs ...int64) error {
	return refreshPerformance(db, "seeds", seedIDs...)
}

func (r *userRepo) RefreshPerformance(ctx context.Context) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return refreshAllPerformance(tx)
	})
}

// RefreshPerformanceFromUsers 仅刷新指定用户及其上级链的业绩缓存。
func (r *userRepo) RefreshPerformanceFromUsers(ctx context.Context, userIDs ...int64) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return refreshSeedAndAncestors(tx, userIDs...)
	})
}
