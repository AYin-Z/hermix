package migrations

import (
	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/repositories"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

// migrate_hermix_boards 落定 Hermix 板块结构（可复现，替代本地即席 SQL）：
//   - 默认节点（id=1）重命名为「社区公告」，normal，置顶
//   - 新增 综合讨论 / 技术交流 / 项目分享（normal）
//   - 新增 互助问答 / 需求广场（qa，复用悬赏采纳闭环）
//
// 幂等：按名称 upsert；默认节点按 id=1 重命名以兼容 zh/en 两种安装语言。
func migrate_hermix_boards() error {
	return sqls.WithTransaction(func(txCtx *sqls.TxContext) error {
		tx := txCtx.Tx
		now := dates.NowTimestamp()

		// 1) 默认节点（id=1，安装语言不同名字不同）→ 社区公告
		if def := repositories.CategoryRepository.Get(tx, 1); def != nil {
			def.Name = "社区公告"
			def.Description = "站点公告、规则与版本更新。"
			def.Type = constants.CategoryTypeNormal
			def.SortNo = 0
			if err := repositories.CategoryRepository.Update(tx, def); err != nil {
				return err
			}
		}

		// PG 修复：早期 migration 以显式 id 插入「默认节点」(id=1)，PostgreSQL 的
		// serial 序列不会随显式 id 推进，仍停在初值。后面按自增创建新板块时序列取到
		// id=1，撞上已存在的默认节点 → t_category_pkey 冲突（MySQL 的 AUTO_INCREMENT
		// 会随手动插入自动前进，故仅 PG 触发）。这里把序列同步到当前 max(id)，使后续
		// 自增从 max(id)+1 开始。幂等，仅对 PostgreSQL 执行。
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec(
				`SELECT setval(pg_get_serial_sequence('t_category', 'id'), ` +
					`GREATEST((SELECT COALESCE(MAX(id), 1) FROM t_category), 1))`,
			).Error; err != nil {
				return err
			}
		}

		// 2) 其余板块：按名称 upsert（存在则更新元数据，不存在则创建）
		boards := []struct {
			Name        string
			Type        constants.CategoryType
			Description string
			SortNo      int
		}{
			{"综合讨论", constants.CategoryTypeNormal, "自由讨论、新手上路与日常交流。", 1},
			{"技术交流", constants.CategoryTypeNormal, "开发、扩展与 AI / Agent 技术探讨。", 2},
			{"项目分享", constants.CategoryTypeNormal, "分享你的开源项目与作品。", 3},
			{"互助问答", constants.CategoryTypeQA, "提出问题、互助解答，可采纳最佳答案。", 4},
			{"需求广场", constants.CategoryTypeQA, "发布悬赏需求，他人接单完成后采纳并支付积分。", 5},
		}

		for _, b := range boards {
			if existing := repositories.CategoryRepository.Take(tx, "name = ?", b.Name); existing != nil {
				existing.Type = b.Type
				existing.Description = b.Description
				existing.SortNo = b.SortNo
				existing.ParentId = 0
				existing.Status = constants.StatusOk
				if err := repositories.CategoryRepository.Update(tx, existing); err != nil {
					return err
				}
				continue
			}

			category := &models.Category{
				ParentId:    0,
				Name:        b.Name,
				Type:        b.Type,
				Description: b.Description,
				SortNo:      b.SortNo,
				Status:      constants.StatusOk,
				CreateTime:  now,
			}
			if err := repositories.CategoryRepository.Create(tx, category); err != nil {
				return err
			}
		}

		return nil
	})
}
