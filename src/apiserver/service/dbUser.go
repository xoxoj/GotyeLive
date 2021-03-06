package service

import (
	"database/sql"
	"errors"
	"fmt"
	"gotye_protocol"
	"strings"

	"github.com/futurez/litego/logger"
	"github.com/futurez/litego/util"
)

func DBCheckUserAccount(username, password string) (userId, headPicId int64, nickname string, sex int8, status_code int) {
	db := SP_MysqlDbPool.GetDBConn()
	var pwd string
	err := db.QueryRow("SELECT user_id, nickname, pwd, headpic_id, sex FROM tbl_users WHERE phone=? OR nickname=?",
		username, username).Scan(&userId, &nickname, &pwd, &headPicId, &sex)
	switch {
	case err == sql.ErrNoRows:
		logger.Infof("DBCheckUserAccount : %s not exists.", username)
		status_code = gotye_protocol.API_USERNAME_NOT_EXISTS_ERROR
		return
	case err != nil:
		logger.Error("DBCheckUserAccount : ", err.Error())
		status_code = gotye_protocol.API_SERVER_ERROR
		return
	}

	if len(pwd) == 0 && len(password) == 0 {
		status_code = gotye_protocol.API_SUCCESS
		return
	}

	if pwd != util.Md5Hash(password) {
		logger.Infof("DBCheckUserAccount : %s password error", username)
		status_code = gotye_protocol.API_LOGIN_PASSWORD_ERROR
		return
	}
	status_code = gotye_protocol.API_SUCCESS
	return
}

func DBGetUserIdByNickname(nickname string) int64 {
	logger.Info("DBGetUserIdByNickname: nickname=", nickname)
	db := SP_MysqlDbPool.GetDBConn()
	var userId int64
	err := db.QueryRow("SELECT user_id FROM tbl_users WHERE nickname=?", nickname).Scan(&userId)
	if err != nil {
		logger.Error("DBGetUserIdByNickname : err=", err.Error())
		return 0
	}
	return userId
}

func DBIsNicknameExists(nickname string) bool {
	logger.Info("DBIsAccountExists : nickname=", nickname)

	db := SP_MysqlDbPool.GetDBConn()
	var count int
	err := db.QueryRow("SELECT count(*) as count FROM tbl_users WHERE nickname=?", nickname).Scan(&count)
	switch {
	case err == sql.ErrNoRows:
		logger.Warn("DBIsPhoneExists : why not row.")
	case err != nil:
		logger.Error("DBIsAccountExists : ", err.Error())
	}
	return count != 0
}

func DBIsPhoneExists(phone string) bool {
	logger.Info("DBIsPhoneExists : phone=", phone)

	db := SP_MysqlDbPool.GetDBConn()
	var count int
	err := db.QueryRow("SELECT count(*) as count FROM tbl_users WHERE phone=?", phone).Scan(&count)
	switch {
	case err == sql.ErrNoRows:
		logger.Warn("DBIsPhoneExists : why not row.")
	case err != nil:
		logger.Error("DBIsPhoneExists : ", err.Error())
	}
	return count != 0
}

//create new user
func DBCreateUserAccount(phone, passwd string) int64 {
	db := SP_MysqlDbPool.GetDBConn()
	res, err := db.Exec("INSERT INTO tbl_users(phone,nickname,pwd) VALUES(?,?,?)", phone, util.RandNickname(), util.Md5Hash(passwd))
	if err != nil {
		logger.Error("DBCreateUserAccount : ", err.Error())
		return -1
	}
	num, _ := res.LastInsertId()
	logger.Info("DBCreateUserAccount : LastInsertId=", num)
	return num
}

func DBModifyUserNickName(userid int64, nickname string) error {
	db := SP_MysqlDbPool.GetDBConn()
	res, err := db.Exec("UPDATE tbl_users SET nickname=? WHERE user_id=?", nickname, userid)
	if err != nil {
		logger.Error("DBModifyUserNickName : ", err.Error())
		return err
	}
	num, _ := res.RowsAffected()
	logger.Info("DBModifyUserNickName : RowsAffected=", num)
	return nil
}

func DBModifyUserInfo(userid int64, sex int8, addr string) error {
	db := SP_MysqlDbPool.GetDBConn()
	var setValue []string
	if sex == 1 || sex == 2 {
		setValue = append(setValue, fmt.Sprintf("sex=%d", sex))
	}
	if len(addr) > 0 {
		setValue = append(setValue, fmt.Sprintf("address='%s'", addr))
	}
	setData := strings.Join(setValue, ",")
	sql := fmt.Sprintf("UPDATE tbl_users SET %s WHERE user_id=%d", setData, userid)
	logger.Info("DBModifyUserInfo : sql=", sql)

	res, err := db.Exec(sql)
	if err != nil {
		logger.Error("DBModifyUserInfo : ", err.Error())
		return err
	}
	num, _ := res.RowsAffected()
	logger.Info("DBModifyUserInfo : RowsAffected=", num)
	return nil
}

func DBGetHeadPicIdByUserId(userid int64) int64 {
	db := SP_MysqlDbPool.GetDBConn()
	var headPicId int64
	err := db.QueryRow("SELECT headpic_id FROM tbl_users WHERE user_id=?", userid).Scan(&headPicId)
	switch {
	case err == sql.ErrNoRows:
		logger.Warn("DBGetHeadPicIdByUserId : why not row.")
	case err != nil:
		logger.Errorf("DBGetHeadPicIdByUserId : userid=%d, err=%s", userid, err.Error())
	default:
		logger.Infof("DBGetHeadPicIdByUserId : user_id=%d, headPicId=%d.", userid, headPicId)
	}
	return headPicId
}

func DBUpdateHeadPicIdByUserId(userId, headPicId int64) error {
	db := SP_MysqlDbPool.GetDBConn()
	res, err := db.Exec("UPDATE tbl_users SET headpic_id=? WHERE user_id=?", headPicId, userId)
	if err != nil {
		logger.Error("DBUpdateHeadPicIdByUserId : ", err.Error())
		return err
	}
	num, _ := res.RowsAffected()
	logger.Info("DBUpdateHeadPicIdByUserId : RowsAffected=", num)
	return nil
}

func DBModifyUserHeadPic(userId int64, headPic []byte) (int64, error) {
	headPicId := DBGetHeadPicIdByUserId(userId)
	db := SP_MysqlDbPool.GetDBConn()
	if headPicId == 0 {
		//add new headPic
		res, err := db.Exec("INSERT INTO tbl_pictures(pic) VALUES(?)", headPic)
		if err != nil {
			logger.Error("DBModifyUserHeadPic : insert into tbl_pictures failed. ", err.Error())
			return 0, err
		}
		num, err := res.LastInsertId()
		if err != nil {
			logger.Error("DBModifyUserHeadPic : get lastinertid failed. ", err.Error())
			return 0, err
		}
		logger.Info("DBModifyUserHeadPic : insert LastInsertId=", num)
		err = DBUpdateHeadPicIdByUserId(userId, num)
		if err != nil {
			logger.Error("DBModifyUserHeadPic : err=", err.Error())
			return 0, err
		}
		return num, nil
	} else {
		res, err := db.Exec("UPDATE tbl_pictures SET pic=? WHERE pic_id=?", headPic, headPicId)
		if err != nil {
			logger.Error("DBModifyUserHeadPic : update tbl_pictures failed, ", err.Error())
			return headPicId, err
		}
		num, _ := res.RowsAffected()
		logger.Info("DBModifyUserHeadPic : Update RowsAffected=", num)
		return headPicId, nil
	}
}

func DBModifyUserPwd(phone, pwd string) error {
	db := SP_MysqlDbPool.GetDBConn()
	res, err := db.Exec("UPDATE tbl_users SET pwd=? WHERE phone=?", util.Md5Hash(pwd), phone)
	if err != nil {
		logger.Error("DBModifyUserPwd : update tbl_pictures failed, ", err.Error())
		return err
	}
	num, _ := res.RowsAffected()
	logger.Info("DBModifyUserPwd : RowsAffected=", num)
	if num == 1 {
		return nil
	} else {
		return errors.New("not exist phone")
	}
}

func DBGetUserHeadPic(picId int64) ([]byte, error) {
	db := SP_MysqlDbPool.GetDBConn()
	var pic []byte
	err := db.QueryRow("SELECT pic FROM tbl_pictures WHERE pic_id=?", picId).Scan(&pic)
	switch {
	case err == sql.ErrNoRows:
		logger.Warn("DBGetUserHeadPic : why not row, picId=", picId)
		return nil, err
	case err != nil:
		logger.Error("DBGetUserHeadPic : ", err.Error())
		return nil, err
	default:
		logger.Info("DBGetUserHeadPic : success get pic_id=", picId)
	}
	return pic, nil
}
