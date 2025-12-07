package sanguo

import (
	"errors"
	"life-online/store/db"
	"time"
)

// GameIdentity 游戏身份表结构
type GameIdentity struct {
	ID          int    `db:"id" json:"id"`
	Description string `db:"description" json:"description"`
	UID         int    `db:"uid" json:"uid"`
	Scope       int    `db:"scope" json:"scope"`
	IsDeleted   int    `db:"is_deleted" json:"is_deleted"`
	CreateTime  int64  `db:"create_time" json:"create_time"`
	UpdateTime  int64  `db:"update_time" json:"update_time"`
}

// CreateGameIdentity 创建游戏身份
func CreateGameIdentity(identity *GameIdentity) (int, error) {
	now := time.Now().Unix()
	identity.CreateTime = now
	identity.UpdateTime = now
	identity.IsDeleted = 0

	query := `INSERT INTO game_identity (description, uid, scope, is_deleted, create_time, update_time) 
			  VALUES (:description, :uid, :scope, :is_deleted, :create_time, :update_time)`

	result, err := db.MainDB.NamedExec(query, identity)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetGameIdentityByID 根据ID查询游戏身份
func GetGameIdentityByID(id int) (*GameIdentity, error) {
	var identity GameIdentity
	query := `SELECT id, description, uid, scope, is_deleted, create_time, update_time 
			  FROM game_identity 
			  WHERE id = ? AND is_deleted = 0`

	err := db.MainDB.Get(&identity, query, id)
	if err != nil {
		return nil, err
	}

	return &identity, nil
}

// UpdateGameIdentity 更新游戏身份
func UpdateGameIdentity(identity *GameIdentity) error {
	if identity.ID == 0 {
		return errors.New("id is required")
	}

	identity.UpdateTime = time.Now().Unix()

	query := `UPDATE game_identity 
			  SET description = :description, uid = :uid, scope = :scope, update_time = :update_time 
			  WHERE id = :id AND is_deleted = 0`

	result, err := db.MainDB.NamedExec(query, identity)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("record not found or already deleted")
	}

	return nil
}

// DeleteGameIdentity 软删除游戏身份
func DeleteGameIdentity(id int) error {
	now := time.Now().Unix()

	query := `UPDATE game_identity 
			  SET is_deleted = 1, update_time = ? 
			  WHERE id = ? AND is_deleted = 0`

	result, err := db.MainDB.Exec(query, now, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("record not found or already deleted")
	}

	return nil
}

// GetGameIdentitiesByScope 通过scope查询并按id倒序
func GetGameIdentitiesByScope(scope int) ([]*GameIdentity, error) {
	var identities []*GameIdentity
	query := `SELECT id, description, uid, scope, is_deleted, create_time, update_time 
			  FROM game_identity 
			  WHERE scope = ? AND is_deleted = 0 
			  ORDER BY id DESC`

	err := db.MainDB.Select(&identities, query, scope)
	if err != nil {
		return nil, err
	}

	return identities, nil
}
