package service

import (
	"errors"

	"feed/dao"
	"feed/model"
)

var ErrTargetUserNotExist = errors.New("target user not exist")

//   - 目标用户不存在 → 哨兵错误（handler 映射 40400）
//   - "不能关注自己(40003)" 文档约定在 handler 层拦（纯参数比较不查库），service 不管
//   - 重复关注幂等：已由 dao.InsertFollow 吞掉唯一键冲突实现，service 无需查重
//   - 列表组装：批量查 user + map 查找表（复用 dao.FindByIDs，别 N+1），
//     组装逻辑参考 service/list.go 的 List
//   - 错误码只属于 handler 层：service 返回哨兵错误

// Follow 关注目标用户（重复关注幂等成功）
func Follow(userID, targetID int64) error {
	// userID 从 token 读取，必然存在，只查目标
	if acc, err := dao.FindByID(targetID); err != nil {
		return err
	} else if acc == nil {
		return ErrTargetUserNotExist
	}
	// 直接插入，重复关注撞 UNIQUE 由 dao 吞掉 → 幂等成功
	return dao.InsertFollow(&model.Follows{FollowerID: userID, FolloweeID: targetID})
}

// Unfollow 取关
func Unfollow(userID, targetID int64) error {
	if acc, err := dao.FindByID(targetID); err != nil {
		return err
	} else if acc == nil {
		return ErrTargetUserNotExist
	}
	// 删除成功或者不存在这条记录都视为成功,返回nil
	return dao.DeleteFollow(&model.Follows{FollowerID: userID, FolloweeID: targetID})
}

// FollowList 关注列表（返回组装好昵称头像的用户）
func FollowList(userID int64) ([]*model.UserBrief, error) {
	if acc, err := dao.FindByID(userID); err != nil {
		return nil, err
	} else if acc == nil {
		return nil, ErrTargetUserNotExist
	}

	ids, err := dao.ListFollows(userID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*model.UserBrief{}, nil
	}
	accounts, err := dao.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	list := make([]*model.UserBrief, len(accounts))
	for i := range accounts {
		list[i] = &model.UserBrief{
			ID:     accounts[i].ID,
			Name:   accounts[i].Name,
			Avatar: accounts[i].Avatar,
		}
	}
	return list, nil
}

// FansList 粉丝列表（同上）
func FansList(userID int64) ([]*model.UserBrief, error) {
	if acc, err := dao.FindByID(userID); err != nil {
		return nil, err
	} else if acc == nil {
		return nil, ErrTargetUserNotExist
	}
	ids, err := dao.ListFollowers(userID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*model.UserBrief{}, nil
	}
	accounts, err := dao.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	list := make([]*model.UserBrief, len(accounts))
	for i := range accounts {
		list[i] = &model.UserBrief{
			ID:     accounts[i].ID,
			Name:   accounts[i].Name,
			Avatar: accounts[i].Avatar,
		}
	}
	return list, nil
}
